package connector

import (
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 8 * time.Second}
