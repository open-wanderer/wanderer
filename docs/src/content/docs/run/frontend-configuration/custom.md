---
title: Add custom pages 
description: How to how add custom pages
---

You can add custom pages to your wanderer instance. The menu items custom pages will be right to the "Lists" menu. You will have to provide these pages **for each** language seperately you want it to be seen.

The menu items will always contain the Filenames of the markdown files (without the .md) including upper/lowercase. For example if your file is called `Weird News.md` the menu item will be called "Weird News".

The Layout will be just like the one from the "About page".

## Installation

Depending on how you installed <span class="-tracking-[0.075em]">wanderer</span> the process of editing the content looks slightly different:
If you already configured the use of a custom `about.md` you might have already the correct configuration:

### Installed via Docker

1. Open your docker-compose.yml file.
2. Find the `volumes` section of the `web` container.
3. Look for the following line (it is commented out by default):
```yaml 
 - ./data/:/app/build/client/md/
 ```
4. Uncomment the line (remove the `#` at the beginning, if present).
5. Make sure the left-hand side (`./data/`) points to a valid directory on your host machine containing the custom subfolders. (There should also be a `about.md` file here)
6. Save your changes and restart the container:
```bash
docker compose restart web
```

 ### Installed from source
1. Navigate to the `web/build/client/md/` directory.
2. Create a subfolder "custom"
3. Create the subfolders per language and files here
4. Save the file.
5. No rebuild is required — the file is loaded dynamically at runtime.

## Creating the folder structure

On the folder mentioned above you need to insert a folder for each language you want to add Menu items for. And in that folder you place the markdown files you want to be displayed.
An example folder structure can look like this (Example is assuming you use Docker folder `data`):
```
data/
├── about.md
├── custom
│   ├── de
│       ├── Kontakt.md
│       ├── Nachrichten.md
│       └── Umfrage.md
│   └── en
│       ├── Contact.md
│       ├── News.md
        └── Survey.md
```

Now the visitor will see the according contents if the langage is set to german (de) or englisch (en)