package controller

import (
	"encoding/json"
	"net/http"
)

type H map[string]any

func JSON(w http.ResponseWriter, data H) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
