package main

import "sync"

var videosLock sync.Mutex
var videos FoundEps = FoundEps{}

var downloadingLock sync.Mutex
var downloading bool = false
