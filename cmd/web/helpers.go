package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

const SEPARATOR = "|"

func (app *application) background(fn func()) {
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		defer func() {
			if err := recover(); err != nil {
				slog.Error(err.(error).Error())
			}
		}()

		fn()
	}()
}

func (app *application) validateInterests(w http.ResponseWriter, r *http.Request) ([]string, error) {
	const MAX_REQUEST_SIZE = 1 * 1024

	r.Body = http.MaxBytesReader(w, r.Body, MAX_REQUEST_SIZE)
	r.ParseForm()

	interests := []string{}
	var interestsErr error

	formInterests := r.Form.Get("interests")

	interestsInp := strings.Split(formInterests, SEPARATOR)

	if formInterests == "" || len(interestsInp) == 0 {
		return interests, interestsErr
	}

	if len(interestsInp) > 3 {
		return interests, fmt.Errorf("maximum of only 3 interests allowed")
	}

	for _, interest := range interestsInp {
		interest = strings.Trim(interest, " ")
		interest = strings.ReplaceAll(interest, SEPARATOR, "_")

		if interestLen := len(interest); interestLen == 0 {
			if interestsErr == nil {
				interestsErr = fmt.Errorf("interest too short (min 1 character)")
			}
		} else if interestLen > 25 {
			if interestsErr == nil {
				interestsErr = fmt.Errorf("interest too long (max 25 characters)")
			}
		}

		interests = append(interests, interest)
	}

	return interests, interestsErr
}

func (app *application) isAuthenticated(r *http.Request) bool {
	isAuthenticated, ok := r.Context().Value(contextKeyIsAuthenticated).(bool)
	if !ok {
		return false
	}

	return isAuthenticated
}
