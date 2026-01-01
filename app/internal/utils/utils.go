package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

func WriteJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func ParseInt64Query(r *http.Request, key string) (int64, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0, errors.New("missing")
	}
	return strconv.ParseInt(v, 10, 64)
}
