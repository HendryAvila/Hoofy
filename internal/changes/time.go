package changes

import "time"

// timeNow is a package-level variable for testability.
// Tests can replace this to control time in assertions.
// Same pattern as pipeline/time.go.
var timeNow = time.Now
