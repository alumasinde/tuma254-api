package ratelimit

import (
	"sync"
	"time"
)

type bucket struct{count int;resetAt time.Time}
type Limiter struct{mu sync.Mutex;buckets map[string]bucket}
func New()*Limiter{return &Limiter{buckets:map[string]bucket{}}}

func(l *Limiter)Allow(key string,limit int,window time.Duration)bool{
	now:=time.Now()
	l.mu.Lock();defer l.mu.Unlock()
	entry,ok:=l.buckets[key]
	if !ok||now.After(entry.resetAt){l.buckets[key]=bucket{count:1,resetAt:now.Add(window)};return true}
	if entry.count>=limit{return false}
	entry.count++;l.buckets[key]=entry;return true
}
