package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

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

func (app *application) extractInterests(w http.ResponseWriter, r *http.Request) ([]string, error) {
	const MAX_REQUEST_SIZE = 1 * 1024
	var interests []string

	r.Body = http.MaxBytesReader(w, r.Body, MAX_REQUEST_SIZE)
	err := r.ParseForm()
	if err != nil {
		return interests, err
	}

	interests = r.Form["interests[]"]

	return interests, nil
}

func (app *application) validateInterests(w http.ResponseWriter, r *http.Request) ([]string, error) {
	interests := []string{}
	var interestsErr error

	interestsRaw, err := app.extractInterests(w, r)
	if err != nil {
		return interests, err
	}

	if len(interestsRaw) == 0 {
		return interests, interestsErr
	}

	if len(interestsRaw) > 3 {
		return interests, fmt.Errorf("maximum of only 3 interests allowed")
	}

	for _, interest := range interestsRaw {
		interest = strings.Trim(interest, " ")

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
