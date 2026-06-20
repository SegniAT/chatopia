package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	MatchTimeout = 60 * time.Second
)

type MatchResult struct {
	Found         bool
	Reason        string
	Partner       *Session
	PartnerMember string
	QueueKey      string
}

var (
	matchScript    *redis.Script
	matchScriptOne sync.Once
)

func getMatchScript() *redis.Script {
	matchScriptOne.Do(func() {
		matchScript = redis.NewScript(matchLuaScript)
	})

	return matchScript
}

var matchLuaScript = `
-- KEYS[1] = queue key (e.g., "queue:text")
-- KEYS[2] = session key (e.g., "session:abc123")
-- ARGV[1] = score (timestamp)
-- ARGV[2] = session ID
--
-- Matching rules:
--   strict:     scan for anyone with a shared interest. no fallback.
--   non-strict: priority 1 = shared interest, priority 2 = any non-strict user (FIFO)

local queueKey = KEYS[1]
local sessionKey = KEYS[2]
local score = tonumber(ARGV[1])
local mySessionID = ARGV[2]

-- Fetch our session data from Redis
local myData = redis.call('GET', sessionKey)
if not myData then
    return cjson.encode({found = false, reason = "session_not_found"})
end

myData = cjson.decode(myData)
if myData['interests'] ~= nil and (myData['interests'] == cjson.null or #myData['interests'] == 0) then
    myData['interests'] = nil
end

local isStrict = myData['is_strict']
local myInterests = myData['interests'] or {}

-- Scan the queue oldest-first
local candidates = redis.call('ZRANGE', queueKey, 0, -1)
local fallback = nil

for i, candidateSessionID in ipairs(candidates) do
    -- self-match guard: remove stale self-entry and skip
    if candidateSessionID == mySessionID then
        redis.call('ZREM', queueKey, candidateSessionID)

    else
        local candidateSessionKey = 'session:' .. candidateSessionID
        local candidateData = redis.call('GET', candidateSessionKey)

        -- stale entry cleanup: session expired, remove from queue
        if not candidateData then
            redis.call('ZREM', queueKey, candidateSessionID)

        else
            local candidate = cjson.decode(candidateData)
            if candidate['interests'] ~= nil and (candidate['interests'] == cjson.null or #candidate['interests'] == 0) then
                candidate['interests'] = nil
            end

            -- check for at least one shared interest
            local hasCommon = false
            for _, myInterest in ipairs(myInterests) do
                for _, theirInterest in ipairs(candidate['interests'] or {}) do
                    if myInterest == theirInterest then
                        hasCommon = true
                        break
                    end
                end
                if hasCommon then break end
            end

            -- interest match found: pair and return
            if hasCommon then
                redis.call('ZREM', queueKey, candidateSessionID)

                local roomID = candidate['session_id'] .. ':' .. mySessionID
                if mySessionID > candidate['session_id'] then
                    roomID = mySessionID .. ':' .. candidate['session_id']
                end

                myData['partner_id'] = candidate['session_id']
                myData['room_id'] = roomID
                redis.call('SET', sessionKey, cjson.encode(myData), 'EX', 86400)

                candidate['partner_id'] = mySessionID
                candidate['room_id'] = roomID
                redis.call('SET', candidateSessionKey, cjson.encode(candidate), 'EX', 86400)

                return cjson.encode({
                    found = true,
                    reason = "interest_match",
                    partner = candidate,
                    partner_member = candidateSessionID
                })
            end

            -- non-strict fallback: remember oldest non-strict candidate
            if not isStrict and not candidate['is_strict'] and fallback == nil then
                fallback = {member = candidateSessionID, key = candidateSessionKey, data = candidate}
            end
        end
    end
end -- for loop

-- no interest match found; non-strict users may fall back to any non-strict user
if not isStrict and fallback ~= nil then
    redis.call('ZREM', queueKey, fallback.member)

    local roomID = fallback.data['session_id'] .. ':' .. mySessionID
    if mySessionID > fallback.data['session_id'] then
        roomID = mySessionID .. ':' .. fallback.data['session_id']
    end

    myData['partner_id'] = fallback.data['session_id']
    myData['room_id'] = roomID
    redis.call('SET', sessionKey, cjson.encode(myData), 'EX', 86400)

    fallback.data['partner_id'] = mySessionID
    fallback.data['room_id'] = roomID
    redis.call('SET', fallback.key, cjson.encode(fallback.data), 'EX', 86400)

    return cjson.encode({
        found = true,
        reason = "non_strict_fallback",
        partner = fallback.data,
        partner_member = fallback.member
    })
end

-- no match found: add ourselves to the queue and wait
redis.call('ZADD', queueKey, score, mySessionID)
redis.call('EXPIRE', queueKey, 3600)
return cjson.encode({found = false, reason = "queue_empty"})
`

func Match(ctx context.Context, redisClient *Client, session *Session) (*MatchResult, error) {
	s := getMatchScript()

	queueKey := QueueKey(session.ChatType)
	sessionKey := SessionKey(session.SessionID)
	score := float64(time.Now().Unix())

	result, err := s.Run(ctx, redisClient.Client, []string{queueKey, sessionKey}, score, session.SessionID).Text()
	if err != nil {
		return nil, fmt.Errorf("match script failed: %w", err)
	}

	var matchResult MatchResult
	if err := json.Unmarshal([]byte(result), &matchResult); err != nil {
		return nil, fmt.Errorf("failed to parse match result: %w, raw: %s", err, result)
	}

	return &matchResult, nil
}
