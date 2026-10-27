package util

import "github.com/doyensec/safeurl"

// Incoming files may use any port, while each connection still applies the
// public-address policy. Clients share this immutable configuration; plugin
// media clients retain their separate port restrictions.
var federatedFileHTTPConfig = func() *safeurl.Config {
	ports := make([]int, 65535)
	for i := range ports {
		ports[i] = i + 1
	}
	return safeurl.GetConfigBuilder().
		SetTimeout(RemoteAttachmentTimeout).
		SetAllowedSchemes("http", "https").
		SetAllowedPorts(ports...).
		SetBlockedIPsCIDR("::/96").
		EnableIPv6(true).
		AllowSendingCredentials(false).
		SetCheckRedirect(publicMediaRedirectPolicy).
		Build()
}()
