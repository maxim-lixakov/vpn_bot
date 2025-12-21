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

func Itoa(x int64) string {
	if x == 0 {
		return "0"
	}
	neg := x < 0
	if neg {
		x = -x
	}
	var b [32]byte
	i := len(b)
	for x > 0 {
		i--
		b[i] = byte('0' + (x % 10))
		x /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func ParseInt64Query(r *http.Request, key string) (int64, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0, errors.New("missing")
	}
	return strconv.ParseInt(v, 10, 64)
}
