package constants

import "time"

const (
    DbPath = "./state/state.db"
    BaseClientURL = "toxiproxy:8474"
    MaxTries = 3
    TimeoutUp = 4000
    TimeoutDown = 4000
    EventPollInterval = 100 * time.Millisecond
    BaseDelay = 100 * time.Millisecond
    HealthCheckPort = ":8000" 
    MaxErrorCount = 15
)

var ProxyConfig = []struct {
    Name     string
    Listen   string
    Upstream string
}{
    {"search_proxy", "0.0.0.0:6000", "search_tool:5000"},
    {"weather_proxy", "0.0.0.0:6001", "weather_tool:5001"},
    {"movie_proxy", "0.0.0.0:6002", "movie_tool:5002"},
    {"calendar_proxy", "0.0.0.0:6003", "calendar_tool:5003"},
    {"calculator_proxy", "0.0.0.0:6004", "calculator_tool:5004"},
    {"message_proxy", "0.0.0.0:6005", "message_tool:5005"},
    {"translator_proxy", "0.0.0.0:6006", "translator_tool:5006"},
}
var Toxics = []string{"toxic_timeout_up", "toxic_timeout_down"}