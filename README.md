# go-limiter

<p align="left">
<a href="https://hits.seeyoufarm.com"><img src="https://hits.seeyoufarm.com/api/count/incr/badge.svg?url=https%3A%2F%2Fgithub.com%2Fgjbae1212%2Fgo-limiter&count_bg=%2379C83D&title_bg=%23555555&icon=go.svg&icon_color=%2308BEB8&title=hits&edge_flat=false"/></a>
<a href="/LICENSE"><img src="https://img.shields.io/badge/license-MIT-GREEN.svg" alt="license"/></a>
</p>

**Go-Limiter** is a rate limiter which can throttle up and back requests in a specified situation.  
**Go-Limiter*** can apply to a custom function that revaluates a current limit of the requests in a specified situation but doesn't get out between min and max limit that you set. 
Revaluate time happens periodically; of course, it supports a custom interval that you set.    
It's based on memory or redis. If you use it in distributed applications, recommend redis.

## Getting Started
```go
package main
import (
    "time"
	"github.com/gjbae1212/go-limiter"
)

func main() {
    cache, _ := limiter.NewMemoryCache() // or limiter.NewRedisCache 
    
    rateLimiter, _ := limiter.NewRateLimiter("service-name", cache,
        limiter.WithEstimatePeriod(10 * time.Minute),
        limiter.WithMinLimit(10),
        limiter.WithMaxLimit(10000),
    )
   
    // ok is true : doesn't exceed limit.
    // ok is false : exceed limit. 
    ok, _ := rateLimiter.Acquire()    
}
```


## LICENSE
This project is following The MIT.
