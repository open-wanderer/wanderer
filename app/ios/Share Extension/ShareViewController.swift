import receive_sharing_intent

// The Share Extension entry point. RSIShareViewController (from
// receive_sharing_intent) handles collecting the shared attachments, writing
// them into the shared App Group container, and redirecting into the host app
// via the ShareMedia-<bundleId> URL scheme.
//
// Keep `shouldAutoRedirect` = true so a single shared file jumps straight into
// Wanderer's import flow without an intermediate compose screen.
class ShareViewController: RSIShareViewController {
    override func shouldAutoRedirect() -> Bool {
        return true
    }
}
