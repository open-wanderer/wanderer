package routes

import "pocketbase/federation"

// instanceInboxHandler is the instance inbox as main.go registers it.
var instanceInboxHandler = federation.NewInstanceInboxHandler(ResolveInboxSender)
