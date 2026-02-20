package pipeline

import "time"

// timeNow is a package-level variable for testability.
// Tests can replace this to control time in assertions.
var timeNow = time.Now
