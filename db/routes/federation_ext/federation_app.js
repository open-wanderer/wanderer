// Alpine component for the federation page. The Alpine CSP build cannot read
// inline scripts or evaluate expressions with optional chaining, so the page
// registers this component by name and keeps all logic here.
// This file is a defer script placed before the Alpine script, so the listener
// exists before Alpine starts.
document.addEventListener('alpine:init', function () {
  Alpine.data('federationApp', function () {
    return {
      token: null,
      authenticated: false,
      email: '',
      peers: [],
      loading: false,
      discoverUrl: '',
      discoverLoading: false,
      discoverError: '',
      connectError: '',
      preview: null,
      rowErrors: {},
      confirmModal: { open: false, peer: null },

      init() {
        var stored = localStorage.getItem('__pb_superusers__/_');
        if (!stored) { return; }
        try {
          var parsed = JSON.parse(stored);
          this.token = parsed.token;
          try {
            var payload = JSON.parse(atob(this.token.split('.')[1]));
            this.email = payload.email || '';
          } catch (_) { }
        } catch (_) { return; }
        if (!this.token) { return; }
        if (this.isTokenExpired(this.token)) { return; }
        this.authenticated = true;
        this.loadPeers();
      },

      isTokenExpired(token) {
        try {
          var payload = JSON.parse(atob(token.split('.')[1]));
          return payload.exp && Date.now() / 1000 > payload.exp;
        } catch (_) { return true; }
      },

      async apiFetch(path, opts) {
        opts = opts || {};
        var res = await fetch(path, Object.assign({}, opts, {
          headers: Object.assign({ 'Authorization': this.token }, opts.headers || {})
        }));
        if (res.status === 401) { this.authenticated = false; return null; }
        return res;
      },

      async loadPeers() {
        this.loading = true;
        this.rowErrors = {};
        try {
          var res = await this.apiFetch('/federation/peers');
          if (res) this.peers = await res.json();
        } finally {
          this.loading = false;
        }
      },

      async discover() {
        this.discoverLoading = true;
        this.discoverError = '';
        this.preview = null;
        try {
          var res = await this.apiFetch('/federation/discover', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ url: this.discoverUrl })
          });
          if (!res) return;
          var data = await res.json();
          if (!res.ok) {
            this.discoverError = this.mapDiscoverError(data.error || 'Discovery failed');
          } else {
            this.preview = {
              actor_id: data.actor_id,
              domain: data.domain,
              version: data.version,
              users: data.user_count,
              trails: data.trail_count
            };
          }
        } finally {
          this.discoverLoading = false;
        }
      },

      mapDiscoverError(raw) {
        if (raw === 'unreachable') return 'Could not reach this URL. Check the address and try again.';
        if (raw === 'not a wanderer instance') return 'This URL does not appear to be a Wanderer instance.';
        if (raw === 'resolves to local instance') return 'This URL resolves to the local instance. You cannot connect to yourself.';
        if (raw === 'already connected') return 'This instance is already connected.';
        return raw;
      },

      async connect() {
        this.loading = true;
        this.connectError = '';
        try {
          var res = await this.apiFetch('/federation/follow', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ actor_id: this.preview.actor_id })
          });
          if (!res) return;
          if (res.ok) {
            this.discoverUrl = '';
            this.preview = null;
            await this.loadPeers();
          } else {
            var data = await res.json();
            this.connectError = 'Connect failed: ' + (data.error || 'unknown error') + '. Try again.';
          }
        } finally {
          this.loading = false;
        }
      },

      cancelPreview() { this.preview = null; this.discoverUrl = ''; },

      async approve(peer) {
        this.loading = true;
        try {
          var res = await this.apiFetch('/federation/approve/' + peer.follow_id, { method: 'POST' });
          if (!res) return;
          if (res.ok) {
            await this.loadPeers();
          } else {
            var data = await res.json();
            this.rowErrors = Object.assign({}, this.rowErrors, { [peer.follow_id]: 'Action failed: ' + (data.error || 'unknown error') + '. Try again.' });
          }
        } finally { this.loading = false; }
      },

      async reject(peer) {
        this.loading = true;
        try {
          var res = await this.apiFetch('/federation/reject/' + peer.follow_id, { method: 'POST' });
          if (!res) return;
          if (res.ok) {
            await this.loadPeers();
          } else {
            var data = await res.json();
            this.rowErrors = Object.assign({}, this.rowErrors, { [peer.follow_id]: 'Action failed: ' + (data.error || 'unknown error') + '. Try again.' });
          }
        } finally { this.loading = false; }
      },

      openDisconnect(peer) {
        this.confirmModal = { open: true, peer };
      },

      closeDisconnect() {
        this.confirmModal = { open: false, peer: null };
      },

      async disconnect() {
        var peer = this.confirmModal.peer;
        this.closeDisconnect();
        this.loading = true;
        try {
          var res = await this.apiFetch('/federation/disconnect/' + peer.follow_id, { method: 'POST' });
          if (!res) return;
          if (res.ok) {
            await this.loadPeers();
          } else {
            var data = await res.json();
            this.rowErrors = Object.assign({}, this.rowErrors, { [peer.follow_id]: 'Action failed: ' + (data.error || 'unknown error') + '. Try again.' });
          }
        } finally { this.loading = false; }
      },

      closeWindow() {
        window.close();
      },

      discoverOnEnter() {
        if (this.discoverUrl.trim() && !this.discoverLoading) { this.discover(); }
      },

      get modalIsWithdraw() {
        var peer = this.confirmModal.peer;
        return !!peer && peer.status === 'pending' && peer.direction === 'outbound';
      },

      get modalIsDisconnect() {
        return !this.modalIsWithdraw;
      },

      get modalDomain() {
        var peer = this.confirmModal.peer;
        return peer ? peer.domain : '';
      },

      get modalConfirmLabel() {
        return this.modalIsWithdraw ? 'Withdraw request' : 'Disconnect';
      }
    };
  });
});
