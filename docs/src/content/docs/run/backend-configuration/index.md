---
title: Backend configuration
description: How to access the PocketBase backend
---

For many configuration options, it is necessary that you are able to access the PocketBase backend. 
PocketBase comes with a handy dashboard that allows you to configure basically everything in the backend.

If you are using docker make sure to forward the internal port 8090 to a public port. 
With the default configuration, the PocketBase admin panel is available at `http://localhost:8090/_/`. 
If this is your first time visiting the panel you will need to create an admin account.
To create backend access navigate to the location of your `docker-compose.yaml` file on the server and type:

```sh
docker compose exec -it db /pocketbase superuser upsert email@example.com myverysecurepassword
```

Via the online dashboard, you will now have access with the user "email@example.com" and the password "myverysecurepassword" to all tables in the backend and can modify the underlying data directly.

For specific configuration guides see:

- [SMTP settings](./smtp/)
- [Auth providers](./auth-providers/)
- [Backup server](./backup-server/)
- [Custom categories](./custom-categories/)
- [Adjust Filesize Limits](./adjust-filesize-limits/)

To learn more about what you can do in the admin dashboard please refer to PocketBase's [documentation](https://pocketbase.io/docs/).
