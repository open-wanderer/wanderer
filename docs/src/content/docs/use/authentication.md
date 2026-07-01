---
title: Authentication
description: Authentication with email/password and OAuth
---

For the majority of <span class="-tracking-[0.075em]">wanderer</span>'s features you need an account to interact with them.

## Email/Username & Password

The quickest way to create an account is by heading over to `/register` and entering a username, a valid email address, and a password of your choice.
After registering you will be redirected to the homepage and can start creating your first trail.

:::note
The username must be at least 3 characters long, the password at least 8.
:::

## OAuth2

Alternatively, <span class="-tracking-[0.075em]">wanderer</span> supports authenticating via OAuth2. The following providers are supported:

- GitHub
- Apple
- Google
- Microsoft
- Yandex
- Facebook
- Instagram
- GitLab
- Bitbucket
- Gitee
- Gitea
- Discord
- Twitter
- Kakao
- VK
- Spotify
- Twitch
- Patreon (v2)
- Strava
- LiveChat
- mailcow
- OpenID Connect

![wanderer OAuth](../../../assets/guides/wanderer_oauth.png)

If your instance offers OAuth logins, the enabled providers appear in <span class="-tracking-[0.075em]">wanderer</span>'s login form. Click the button, authorize <span class="-tracking-[0.075em]">wanderer</span>, and wait for the authentication to finish. You are now logged in and can use <span class="-tracking-[0.075em]">wanderer</span> like any other user.

For instructions on enabling OAuth2 providers for your own instance, see the [OAuth2 setup guide](/run/backend-configuration/oauth2/).

## API Tokens

API tokens allow external tools and automated processes to interact with your <span class="-tracking-[0.075em]">wanderer</span> account without requiring your login credentials.

:::danger
API tokens grant full access to your account. Do not share them with untrusted parties.
:::

To manage your tokens:
1. Log in to your <span class="-tracking-[0.075em]">wanderer</span> instance.
2. Navigate to **Settings** > **Account** > **API Tokens**.
3. Click **Generate new token**, provide a descriptive name and optionally an expiration date and click "Save".
4. **Copy the token immediately.** For security, it will not be shown again.


## Forgot your password?
<span class="-tracking-[0.075em]">wanderer</span> offers the option to send password reset emails in case a user forgets his password.
You can click the "Forgot password" link in the login form. After requesting the reset the user will receive an email with a unique link to reset their password.