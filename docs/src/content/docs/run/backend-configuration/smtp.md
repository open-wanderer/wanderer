---
title: SMTP settings
description: Configure email notifications
redirectFrom:
  - /run/backend-configuration/#configure-smtp-settings
---

<span class="-tracking-[0.075em]">wanderer</span> can send email notifications to users (e.g. when a user gains a new follower). 
This is also relevant to send password reset notifications. 
To enable sending email, you need to configure your SMPT settings in PocketBase.

![Pocketbase Mail Settings](../../../../assets/guides/pocketbase_mail_settings.png)

In the pocketbase admin panel go to Settings -> Mail settings an enable "Use SMTP mail server". 
Enter the details of your SMTP server and send a test email to ensure your configuration is correct. 
On the same page you can also adjust the email template of the password reset email.

Alternatively, you can set these options via the respective [environment variables](/run/environment-configuration/#pocketbase).
