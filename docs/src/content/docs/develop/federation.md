---
title: Federation
description: Technical documentation of federation in wanderer
---

wanderer is a federated trail-sharing platform built on ActivityPub. It enables users to publish trails, follow other explorers across instances, and interact with content such as comments, lists, and summit logs. All user-generated content in wanderer—whether it's a trail, a list, a comment, or a summit log—is modeled as a `Note` object in ActivityPub, adhering to a consistent structure for federation.

This technical documentation provides a detailed overview of how federation works in wanderer, including the types of objects exchanged, the structure of those objects, and how interactions such as mentions, likes, and follows are processed across instances.

Below, you’ll find examples of the different JSON representations used in federated communication. These illustrate how <span class="-tracking-[0.075em]">wanderer</span> encodes and interprets core actions and content as standardized `Note` objects.

## Loading remote trails and lists

Full synchronization accepts a document of at most 32 MiB and 1,000,000 JSON tokens. Token counting includes object keys, scalar values and opening and closing delimiters. Documents above 500,000 tokens reserve both processing slots; smaller documents may use one slot each. Network waits use separate limits.

A fresh copy whose document exceeds either limit returns HTTP 422 with a clear message and an `origin_url` in its error data. The web interface offers a link to the original trail or list. A completed copy remains readable if a later refresh exceeds a limit.

## Context

```json
"@context":[
    "https://www.w3.org/ns/activitystreams"
]
```
The context is identical for all activities and objects.

## Actors
An actor represents a user of <span class="-tracking-[0.075em]">wanderer</span> in a federated context.

### Person

```json
{
    "id": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
    "type": "Person",
    "inbox": "https://demo.wanderer.to/api/v1/activitypub/user/demo/inbox",
    "outbox": "https://demo.wanderer.to/api/v1/activitypub/user/demo/outbox",
    "summary": "Born the day we installed the site.",
    "name": "demo",
    "preferredUsername": "demo",
    "followers": "https://demo.wanderer.to/api/v1/activitypub/user/demo/followers",
    "following": "https://demo.wanderer.to/api/v1/activitypub/user/demo/following",
    "url": "https://demo.wanderer.to/profile/@demo",
    "published": "2025-05-05T15:07:59.943Z",
    "icon": {
        "type": "Image",
        "url": "https://demo.wanderer.to/api/v1/files/users/26b1si1344ficl6/wlezq_um7vh2722q.jpg"
    },
    "publicKey": {
        "id": "https://demo.wanderer.to/api/v1/activitypub/user/demo#main-key",
        "owner": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
        "publicKeyPem": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAw0xyaRWP5X955bwSnUbr\nmwEF/2Fdmn5nlRRmEvej1BR0oBcPMVPYrrK4sz37mrAJ7Wbmg4KjmSDEROD4sApr\nM5FmKeU1OBsV2O3bL1DSW/8PXaf4JQRgl0AO+LiSAd7A/GO0viAzJXyJT4Rpaamf\n8Naclh7YR5E4JXrsjahPEWtUWcQ4g8Yhc6n2ptQ33ACI7Q1R3+U7q1tMaRCKAbdT\nbRahzqGs3iSxV+FjnsMR109KqDQJDMjwRB11USJTA4/nMpV6w8RS+171xNHl12Sg\nGpiuusmXMYYuoECdKDtLY7AsntusYMzXUjPzKfE+5EqPmIj5OTbg3A24p9hWIv5s\nmwIDAQAB\n-----END PUBLIC KEY-----\n"
    }
}
```

### Outbox

Paginated outbox of an actor.

```json
{
    "type": "OrderedCollectionPage",
    "first": "https://demo.wanderer.to/api/v1/activitypub/user/demo/outbox?page=1",
    "next": "https://demo.wanderer.to/api/v1/activitypub/user/demo/outbox?page=2",
    "partOf": "https://demo.wanderer.to/api/v1/activitypub/user/demo/outbox",
    "totalItems": 23,
    "orderedItems": [
        {
            "id": "https://demo.wanderer.to/api/v1/activitypub/activity/ecy96j9vpke00hr",
            "actor": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
            "type": "Create",
            "to": [
                "https://www.w3.org/ns/activitystreams#Public"
            ],
            "cc": [
                "https://social.tchncs.de/users/flomp/inbox",
                "https://demo.wanderer.to/api/v1/activitypub/user/demo/inbox"
            ],
            "published": "2025-06-20 19:41:53.504Z",
            "object": {
                "id": "https://demo.wanderer.to/api/v1/comment/htm169g4b2i48fc",
                "type": "Note",
                "content": "<p><a href=\"/profile/@flomp@social.tchncs.de\" class=\"mention\" rel=\"nofollow\">@flomp@social.tchncs.de</a> </p><p>Wow! What a beautiful trail!</p>",
                "attributedTo": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
                "inReplyTo": "https://demo.wanderer.to/api/v1/trail/2ce3af7a2e80f52",
                "tag": [
                    {
                        "id": "https://social.tchncs.de/users/flomp",
                        "type": "Mention",
                        "name": "@flomp@social.tchncs.de",
                        "href": "https://social.tchncs.de/users/flomp"
                    }
                ],
                "published": "2025-06-20T19:41:53Z"
            }
        }
    ]
}
```

### Followers

Paginated collection of followers of an actor.

```json
{
    "type": "OrderedCollectionPage",
    "first": "https://demo.wanderer.to/api/v1/activitypub/user/demo/followers?page=1",
    "partOf": "https://demo.wanderer.to/api/v1/activitypub/user/demo/followers",
    "totalItems": 3,
    "orderedItems": [
        "https://social.tchncs.de/users/flomp",
        "https://trails.magdeburg.jetzt/api/v1/activitypub/user/momar",
        "https://darmstadt.social/users/stormii"
    ]
}
```

### Following

Paginated collection of actors being followed by an actor.

```json
{
    "type": "OrderedCollectionPage",
    "first": "https://demo.wanderer.to/api/v1/activitypub/user/demo/following?page=1",
    "partOf": "https://demo.wanderer.to/api/v1/activitypub/user/demo/following",
    "totalItems": 3,
    "orderedItems": [
        "https://social.tchncs.de/users/milan",
        "https://social.tchncs.de/users/flomp",
        "https://trails.tchncs.de/api/v1/activitypub/user/milan"
    ]
}
```

## Objects

### Trail

Represents a trail with various metadata like description, photos, elevation data etc. 

:::note
Waypoints, comments and summit logs are not part of a federated trail object. They are instead fetched on demand from the source instance when requesting a trail.
:::

```json
{
    "id": "https://demo.wanderer.to/api/v1/trail/2ce3af7a2e80f52",
    "type": "Note",
    "name": "12 days in the Zugspitz region on the peak hiking trail",
    "content": "<h1>12 days in the Zugspitz region on the peak hiking trail</h1><p><a href=\"/profile/@flomp@social.tchncs.de\" class=\"mention\" rel=\"nofollow\">@flomp@social.tchncs.de</a> </p><p><a href=\"https://demo.wanderer.to/trail/view/@demo/2ce3af7a2e80f52\">https://demo.wanderer.to/trail/view/@demo/2ce3af7a2e80f52</a></p>",
    "attachment": [
        {
            "type": "Image",
            "mediaType": "image/jpeg",
            "url": "https://demo.wanderer.to/api/v1/files/trails/2ce3af7a2e80f52/route_ldv172t0my.webp"
        },
        {
            "type": "Document",
            "mediaType": "application/xml+gpx",
            "url": "https://demo.wanderer.to/api/v1/files/trails/2ce3af7a2e80f52/12_days_in_the_zugspitz_region_on_the_peak_hiking_trail_5ts04zgsuk.gpx"
        }
    ],
    "attributedTo": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
    "location": {
        "type": "Place",
        "name": "Murnau am Staffelsee, Bayern, Deutschland",
        "latitude": 47.678592,
        "longitude": 11.196068
    },
    "tag": [
        {
            "type": "Note",
            "name": "category",
            "content": "Biking"
        },
        {
            "type": "Note",
            "name": "difficulty",
            "content": "easy"
        },
        {
            "type": "Note",
            "name": "elevation_gain",
            "content": "8902.000000m"
        },
        {
            "type": "Note",
            "name": "elevation_loss",
            "content": "8906.000000m"
        },
        {
            "type": "Note",
            "name": "distance",
            "content": "202403.824936m"
        },
        {
            "type": "Note",
            "name": "duration",
            "content": "4037.183333m"
        },
        {
            "id": "https://social.tchncs.de/users/flomp",
            "type": "Mention",
            "name": "@flomp@social.tchncs.de",
            "href": "https://social.tchncs.de/users/flomp"
        },
        {
            "type": "Note",
            "name": "tag",
            "content": "My awesome tag"
        }
    ],
    "url": "https://demo.wanderer.to/trail/view/@demo/2ce3af7a2e80f52",
    "published": "2025-06-17T21:40:02Z",
    "startTime": "2025-06-14T00:00:00Z"
}
```

### Summit log

Represents a summit log that is attached to a trail. The trail is referenced in the "InReplyTo" field. It contains very similar metadata to a trail object.

```json
{
  "id": "https://demo.wanderer.to/api/v1/summit-log/0l889g7nbju9ic2",
  "type": "Note",
  "content": "<p>Hello World! This is a summit log!</p><p><a href=\"/profile/@flomp@social.tchncs.de\" class=\"mention\" rel=\"nofollow\">@flomp@social.tchncs.de</a> </p>",
  "attachment": [
    {
      "type": "Image",
      "mediaType": "image/jpeg",
      "url": "https://demo.wanderer.to/api/v1/files/summit_logs/0l889g7nbju9ic2/walchensee_heimgarten_fahrenbergkopf_herzogstand_2020_10_25_loncv4fixp.jpg"
    },
    {
      "type": "Document",
      "mediaType": "application/xml+gpx",
      "url": "https://demo.wanderer.to/api/v1/files/summit_logs/0l889g7nbju9ic2/12_days_in_the_zugspitz_region_on_the_peak_hiking_trail_8mw5gysia0.gpx"
    }
  ],
  "attributedTo": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
  "inReplyTo": "https://demo.wanderer.to/api/v1/trail/2ce3af7a2e80f52",
  "tag": [
    {
      "type": "Note",
      "name": "elevation_gain",
      "content": "6343.200000m"
    },
    {
      "type": "Note",
      "name": "elevation_loss",
      "content": "6347.400000m"
    },
    {
      "type": "Note",
      "name": "distance",
      "content": "202403.824936m"
    },
    {
      "type": "Note",
      "name": "duration",
      "content": "242231.000000m"
    },
    {
      "id": "https://social.tchncs.de/users/flomp",
      "type": "Mention",
      "name": "@flomp@social.tchncs.de",
      "href": "https://social.tchncs.de/users/flomp"
    }
  ],
  "url": "https://demo.wanderer.to/trail/view/@demo/2ce3af7a2e80f52",
  "published": "2025-06-20T19:38:19Z",
  "startTime": "2025-06-20T00:00:00Z"
}
```

### Comment

A comment attached to a trail. The trail is referenced in the "InReplyTo" field. Contains only text.

```json
{
  "id": "https://demo.wanderer.to/api/v1/comment/htm169g4b2i48fc",
  "type": "Note",
  "content": "<p><a href=\"/profile/@flomp@social.tchncs.de\" class=\"mention\" rel=\"nofollow\">@flomp@social.tchncs.de</a> </p><p>Wow! What a beautiful trail!</p>",
  "attributedTo": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
  "inReplyTo": "https://demo.wanderer.to/api/v1/trail/2ce3af7a2e80f52",
  "tag": [
    {
      "id": "https://social.tchncs.de/users/flomp",
      "type": "Mention",
      "name": "@flomp@social.tchncs.de",
      "href": "https://social.tchncs.de/users/flomp"
    }
  ],
  "published": "2025-06-20T19:41:53Z"
}
```

### List

A collection of trails.

```json
{
  "id": "https://demo.wanderer.to/api/v1/list/65i686yf6u3b394",
  "type": "Note",
  "name": "My Awesome List",
  "content": "<p>With my awesome description.</p><p><a href=\"https://demo.wanderer.to/lists/@demo/65i686yf6u3b394\">https://demo.wanderer.to/lists/@demo/65i686yf6u3b394</a></p>",
  "attachment": [
    {
      "type": "Image",
      "mediaType": "image/jpeg",
      "url": "https://demo.wanderer.to/api/v1/files/lists/65i686yf6u3b394/walchensee_heimgarten_fahrenbergkopf_herzogstand_2020_10_25_m1ubtj7rwk.jpg"
    }
  ],
  "attributedTo": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
  "url": "https://demo.wanderer.to/lists/@demo/65i686yf6u3b394",
  "published": "2025-05-18T22:03:19Z"
}
```

## Activities

### Create or Update trail

Issued whenever a trail is created or updated. Broadcasted to all followers and all mentions. Editing a previously created trail will broadcast an identical activity, except the `type` being `Update`. The `object` is a [Trail](#trail).

```json
{
    "id": "https://demo.wanderer.to/api/v1/activitypub/activity/wqt6poxjevq9oax",
    "actor": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
    "type": "Create",
    "to": [
        "https://www.w3.org/ns/activitystreams#Public"
    ],
    "cc": [
        "https://demo.wanderer.to/api/v1/activitypub/user/demo/followers",
        "https://social.tchncs.de/users/flomp/inbox"
    ],
    "published": "2025-06-15 14:56:38.800Z",
    "object": {}
}
```

### Create or Update summit log

Issued whenever a summit log is created or updated. Broadcasted to the trail author, the author's followers and all mentions. Editing a previously created summit log will broadcast an identical activity, except the `type` being `Update`. The `object` is a [Summit Log](#summit-log).

```json
{
    "id": "https://demo.wanderer.to/api/v1/activitypub/activity/i31uc0lki3crxwm",
    "actor": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
    "to": "https://www.w3.org/ns/activitystreams#Public",
    "type": "Create",
    "cc": [
        "https://demo.wanderer.to/api/v1/activitypub/user/demo/followers",
        "https://social.tchncs.de/users/flomp/inbox"
    ],
    "published": "2025-06-20 19:38:19.978Z",
    "object": {}
}
```

### Create or Update comment

Issued whenever a comment is created or updated. Broadcasted to the trail's author and all mentions. Editing a previously created comment will broadcast an identical activity, except the `type` being `Update`. The `object` is a [Comment](#comment).

```json
{
    "id": "https://demo.wanderer.to/api/v1/activitypub/activity/ecy96j9vpke00hr",
    "actor": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
    "type": "Create",
    "to": [
        "https://www.w3.org/ns/activitystreams#Public"
    ],
    "cc": [
        "https://social.tchncs.de/users/flomp/inbox",
        "https://demo.wanderer.to/api/v1/activitypub/user/demo/inbox"
    ],
    "published": "2025-06-20 19:41:53.504Z",
    "object": {}
}
```


### Create or Update list

Issued whenever a list is created or updated. Broadcasted to all followers. Editing a previously created list will broadcast an identical activity, except the `type` being `Update`. The `object` is a [List](#list).

```json
{
    "id": "https://demo.wanderer.to/api/v1/activitypub/activity/zq30he84ng9of67",
    "actor": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
    "type": "Create",
    "to": [
        "https://www.w3.org/ns/activitystreams#Public"
    ],
    "cc": [
        "https://demo.wanderer.to/api/v1/activitypub/user/demo/followers",
    ],
    "published": "2025-06-20 19:51:37.079Z",
    "object": {}
}
```

### Follow user

Each actor in <span class="-tracking-[0.075em]">wanderer</span> can be followed. The actor being followed will immediately send back an `Accept` activity. Future public trails and lists published by the actor being followed will be broadcasted to the following actors inbox.

```json
{
    "id": "https://demo.wanderer.to/api/v1/activitypub/activity/ika3t06qjyvlx72",
    "actor": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
    "type": "Follow",
    "to": null,
    "cc": null,
    "published": "2025-06-01 07:23:56.517Z",
    "object": "https://social.tchncs.de/users/flomp"
}
```

### Accept follow

Automatically send by an actor as a response upon receiving a `Follow` activity.

```json
{
    "id": "https://demo.wanderer.to/api/v1/activitypub/activity/9jpjjvvi79ayp9d",
    "actor": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
    "type": "Accept",
    "to": null,
    "cc": null,
    "published": "2025-06-17 19:48:29.417Z",
    "object": {
        "id": "https://social.tchncs.de/8f702e81-9f85-419f-8e45-c44c8b6d8365",
        "type": "Follow",
        "actor": "https://social.tchncs.de/users/flomp",
        "object": "https://demo.wanderer.to/api/v1/activitypub/user/demo"
    }
}
```

### Undo follow

An unfollow is represented by an `Undo` activity with the original follow as its `object`.

```json
{
    "id": "https://demo.wanderer.to/api/v1/activitypub/activity/n3n7ka5msa3il84",
    "actor": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
    "type": "Undo",
    "to": null,
    "cc": null,
    "published": "2025-06-20 20:14:37.107Z",
    "object": {
        "id": "https://demo.wanderer.to/api/v1/activitypub/activity/ika3t06qjyvlx72",
        "type": "Follow",
        "actor": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
        "object": "https://social.tchncs.de/users/flomp"
    }
}
```

### Like trail

A like for a trail.

```json
{
    "id": "https://demo.wanderer.to/api/v1/activitypub/activity/jjlcgm0il3jy2y7",
    "actor": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
    "type": "Like",
    "to": null,
    "cc": null,
    "published": "2025-06-18 18:48:55.300Z",
    "object": "https://demo.wanderer.to/api/v1/trail/23fd1747a29c3af"
}
```

### Undo like trail

Removing a like from a previously liked trail. 

```json
{
    "id": "https://demo.wanderer.to/api/v1/activitypub/activity/jjlcgm0il3jy2y7",
    "actor": "https://demo.wanderer.to/api/v1/activitypub/user/demo",
    "type": "Undo",
    "to": null,
    "cc": null,
    "published": "2025-06-18 18:48:55.300Z",
    "object": "https://demo.wanderer.to/api/v1/trail/23fd1747a29c3af"
}
```