package main

import "sync"

var videosLock sync.Mutex
var videos FoundEps = FoundEps{}
