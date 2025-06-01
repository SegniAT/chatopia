package main

import (
	"fmt"
	"net/http"
	"strings"
)

func (app *application) extractInterests(w http.ResponseWriter, r *http.Request) (bool, []string, error) {
	const MAX_REQUEST_SIZE = 1 * 1024
	var interests []string
	var isStrict bool

	r.Body = http.MaxBytesReader(w, r.Body, MAX_REQUEST_SIZE)
	err := r.ParseForm()
	if err != nil {
		return isStrict, interests, err
	}

	interests = r.Form["interests[]"]
	isStrict = r.Form.Get("isStrict") == "true"

	return isStrict, interests, nil
}

func (app *application) validateInterests(w http.ResponseWriter, r *http.Request) (bool, []string, error) {
	interests := []string{}
	var interestsErr error
	var isStrict bool

	isStrict, interestsRaw, err := app.extractInterests(w, r)
	if err != nil {
		return isStrict, interests, err
	}

	if len(interestsRaw) == 0 {
		return isStrict, interests, interestsErr
	}

	if len(interestsRaw) > 3 {
		return isStrict, interests, fmt.Errorf("maximum of only 3 interests allowed")
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

	return isStrict, interests, interestsErr
}

func (app *application) isAuthenticated(r *http.Request) bool {
	isAuthenticated, ok := r.Context().Value(contextKeyIsAuthenticated).(bool)
	if !ok {
		return false
	}

	return isAuthenticated
}
