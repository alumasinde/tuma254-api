package ratelimit

import (
	"sync"
	"time"
)

type bucket struct{ count int; resetAt time.Time }
type Limiter struct{ mu sync.Mutex; buckets map[string]bucket; lastCleanup time.Time }

func New()*Limiter{return &Limiter{buckets:map[string]bucket{}}}

func(l *Limiter)Allow(key string,limit int,window time.Duration)bool{
	now:=time.Now()
	l.mu.Lock(); defer l.mu.Unlock()
	if now.Sub(l.lastCleanup) >= window {
		for k,v:=range l.buckets { if !now.Before(v.resetAt) { delete(l.buckets,k) } }
		l.lastCleanup=now
	}
	entry,ok:=l.buckets[key]
	if !ok||!now.Before(entry.resetAt){l.buckets[key]=bucket{count:1,resetAt:now.Add(window)};return true}
	if entry.count>=limit{return false}
	entry.count++;l.buckets[key]=entry;return true
}
