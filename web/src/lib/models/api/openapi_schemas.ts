/**
 * @swagger
 * components:
 *   schemas:
 *     APIToken:
 *       type: object
 *       required:
 *         - id
 *         - name
 *         - user
 *       properties:
 *         id:
 *           type: string
 *           description: Token ID (15 chars)
 *         name:
 *           type: string
 *           description: Token name
 *         user:
 *           type: string
 *           description: User ID that owns this token
 *         expiration:
 *           type: string
 *           format: date-time
 *           description: Optional expiration date
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     APITokenInput:
 *       type: object
 *       required:
 *         - name
 *         - user
 *       properties:
 *         name:
 *           type: string
 *           minLength: 1
 *           description: Token name
 *         expiration:
 *           type: string
 *           format: date-time
 *           description: Optional expiration date (must be in future)
 *         user:
 *           type: string
 *           description: User ID (15 chars)
 *
 *     User:
 *       type: object
 *       required:
 *         - id
 *         - username
 *         - email
 *       properties:
 *         id:
 *           type: string
 *           description: User ID (15 chars)
 *         username:
 *           type: string
 *           description: Username (3+ chars, alphanumeric with dots)
 *         email:
 *           type: string
 *           format: email
 *           description: User email address
 *         avatar:
 *           type: string
 *           description: Avatar file path
 *         verified:
 *           type: boolean
 *           description: Email verification status
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     UserCreateInput:
 *       type: object
 *       required:
 *         - username
 *         - email
 *         - password
 *       properties:
 *         username:
 *           type: string
 *           minLength: 3
 *           pattern: '^\w[\w\.]*$'
 *           description: Username (3+ chars, starts with word char)
 *         email:
 *           type: string
 *           format: email
 *           description: User email address
 *         password:
 *           type: string
 *           minLength: 8
 *           maxLength: 72
 *           description: Password (8-72 chars)
 *         passwordConfirm:
 *           type: string
 *           description: Password confirmation (must match password)
 *
 *     UserUpdateInput:
 *       type: object
 *       properties:
 *         username:
 *           type: string
 *           minLength: 3
 *           pattern: '^\w[\w\.]*$'
 *         email:
 *           type: string
 *           format: email
 *         password:
 *           type: string
 *           minLength: 8
 *           maxLength: 72
 *         oldPassword:
 *           type: string
 *           description: Required when changing password
 *         passwordConfirm:
 *           type: string
 *           description: Must match password
 *
 *     Tag:
 *       type: object
 *       required:
 *         - id
 *         - name
 *       properties:
 *         id:
 *           type: string
 *           description: Tag ID (15 chars)
 *         name:
 *           type: string
 *           description: Tag name
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     TagInput:
 *       type: object
 *       required:
 *         - name
 *       properties:
 *         name:
 *           type: string
 *           description: Tag name
 *
 *     TagUpdateInput:
 *       type: object
 *       properties:
 *         name:
 *           type: string
 *           description: Tag name
 *
 *     Trail:
 *       type: object
 *       required:
 *         - id
 *         - name
 *         - author
 *         - public
 *       properties:
 *         id:
 *           type: string
 *           description: Trail ID (15 chars)
 *         name:
 *           type: string
 *           description: Trail name
 *         description:
 *           type: string
 *         location:
 *           type: string
 *         author:
 *           type: string
 *           description: Author user ID
 *         date:
 *           type: string
 *           format: date
 *         public:
 *           type: boolean
 *         difficulty:
 *           type: string
 *           enum: [easy, moderate, difficult]
 *         lat:
 *           type: number
 *           minimum: -90
 *           maximum: 90
 *         lon:
 *           type: number
 *           minimum: -180
 *           maximum: 180
 *         distance:
 *           type: number
 *           description: Distance in meters
 *         elevation_gain:
 *           type: number
 *           description: Elevation gain in meters
 *         elevation_loss:
 *           type: number
 *           description: Elevation loss in meters
 *         duration:
 *           type: number
 *           description: Duration in seconds
 *         photos:
 *           type: array
 *           items:
 *             type: string
 *         thumbnail:
 *           type: integer
 *           description: Index of thumbnail photo
 *         like_count:
 *           type: integer
 *           default: 0
 *         category:
 *           type: string
 *           description: Category ID (15 chars)
 *         tags:
 *           type: array
 *           items:
 *             type: string
 *         gpx:
 *           type: string
 *           description: GPX file path
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     TrailCreateInput:
 *       type: object
 *       required:
 *         - name
 *         - author
 *         - public
 *       properties:
 *         id:
 *           type: string
 *           description: Optional custom ID (15 chars)
 *         name:
 *           type: string
 *           minLength: 1
 *         author:
 *           type: string
 *           description: Author user ID (15 chars)
 *         description:
 *           type: string
 *         location:
 *           type: string
 *         date:
 *           type: string
 *           format: date
 *         public:
 *           type: boolean
 *         difficulty:
 *           type: string
 *           enum: [easy, moderate, difficult]
 *         lat:
 *           type: number
 *           minimum: -90
 *           maximum: 90
 *         lon:
 *           type: number
 *           minimum: -180
 *           maximum: 180
 *         distance:
 *           type: number
 *         elevation_gain:
 *           type: number
 *         elevation_loss:
 *           type: number
 *         duration:
 *           type: number
 *         photos:
 *           type: array
 *           items:
 *             type: string
 *           default: []
 *         thumbnail:
 *           type: integer
 *         like_count:
 *           type: integer
 *           default: 0
 *         category:
 *           type: string
 *         tags:
 *           type: array
 *           items:
 *             type: string
 *           default: []
 *         gpx:
 *           type: string
 *
 *     TrailUpdateInput:
 *       type: object
 *       properties:
 *         name:
 *           type: string
 *         description:
 *           type: string
 *         location:
 *           type: string
 *         date:
 *           type: string
 *           format: date
 *         public:
 *           type: boolean
 *         difficulty:
 *           type: string
 *           enum: [easy, moderate, difficult]
 *         lat:
 *           type: number
 *           minimum: -90
 *           maximum: 90
 *         lon:
 *           type: number
 *           minimum: -180
 *           maximum: 180
 *         distance:
 *           type: number
 *         elevation_gain:
 *           type: number
 *         elevation_loss:
 *           type: number
 *         duration:
 *           type: number
 *         photos:
 *           type: array
 *           items:
 *             type: string
 *         'photos-':
 *           type: string
 *           description: Remove photo by name
 *         'photos+':
 *           type: string
 *           description: Add photo by name
 *         thumbnail:
 *           type: integer
 *         like_count:
 *           type: integer
 *           default: 0
 *         category:
 *           type: string
 *         tags:
 *           type: array
 *           items:
 *             type: string
 *         gpx:
 *           type: string
 *
 *     Comment:
 *       type: object
 *       required:
 *         - id
 *         - text
 *         - author
 *         - trail
 *       properties:
 *         id:
 *           type: string
 *           description: Comment ID (15 chars)
 *         text:
 *           type: string
 *         author:
 *           type: string
 *           description: Author user ID (15 chars)
 *         trail:
 *           type: string
 *           description: Trail ID (15 chars)
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     CommentInput:
 *       type: object
 *       required:
 *         - text
 *         - author
 *         - trail
 *       properties:
 *         text:
 *           type: string
 *         author:
 *           type: string
 *           description: Author user ID (15 chars)
 *         trail:
 *           type: string
 *           description: Trail ID (15 chars)
 *
 *     CommentUpdateInput:
 *       type: object
 *       properties:
 *         text:
 *           type: string
 *
 *     SummitLog:
 *       type: object
 *       required:
 *         - id
 *         - date
 *         - author
 *       properties:
 *         id:
 *           type: string
 *           description: Summit log ID (15 chars)
 *         date:
 *           type: string
 *           format: date
 *         text:
 *           type: string
 *         author:
 *           type: string
 *           description: Author user ID (15 chars)
 *         trail:
 *           type: string
 *           description: Associated trail ID
 *         gpx:
 *           type: string
 *         distance:
 *           type: number
 *         elevation_gain:
 *           type: number
 *         elevation_loss:
 *           type: number
 *         duration:
 *           type: number
 *         photos:
 *           type: array
 *           items:
 *             type: string
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     SummitLogInput:
 *       type: object
 *       required:
 *         - date
 *         - author
 *       properties:
 *         id:
 *           type: string
 *           description: Optional custom ID (15 chars)
 *         date:
 *           type: string
 *           format: date
 *         text:
 *           type: string
 *         author:
 *           type: string
 *           description: Author user ID (15 chars)
 *         trail:
 *           type: string
 *           description: Trail ID (15 chars)
 *         gpx:
 *           type: string
 *         distance:
 *           type: number
 *         elevation_gain:
 *           type: number
 *         elevation_loss:
 *           type: number
 *         duration:
 *           type: number
 *         photos:
 *           type: array
 *           items:
 *             type: string
 *           default: []
 *
 *     SummitLogUpdateInput:
 *       type: object
 *       properties:
 *         date:
 *           type: string
 *           format: date
 *         text:
 *           type: string
 *         gpx:
 *           type: string
 *         distance:
 *           type: number
 *         elevation_gain:
 *           type: number
 *         elevation_loss:
 *           type: number
 *         duration:
 *           type: number
 *         photos:
 *           type: array
 *           items:
 *             type: string
 *         'photos-':
 *           type: string
 *           description: Remove photo by name
 *         'photos+':
 *           type: string
 *           description: Add photo by name
 *
 *     Waypoint:
 *       type: object
 *       required:
 *         - id
 *         - lat
 *         - lon
 *         - author
 *       properties:
 *         id:
 *           type: string
 *           description: Waypoint ID (15 chars)
 *         name:
 *           type: string
 *         description:
 *           type: string
 *         lat:
 *           type: number
 *           minimum: -90
 *           maximum: 90
 *         lon:
 *           type: number
 *           minimum: -180
 *           maximum: 180
 *         distance_from_start:
 *           type: number
 *           description: Distance from trail start in meters
 *         icon:
 *           type: string
 *           description: Icon identifier
 *         author:
 *           type: string
 *           description: Author user ID (15 chars)
 *         photos:
 *           type: array
 *           items:
 *             type: string
 *         trail:
 *           type: string
 *           description: Trail ID (15 chars)
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     WaypointInput:
 *       type: object
 *       required:
 *         - lat
 *         - lon
 *         - author
 *       properties:
 *         id:
 *           type: string
 *           description: Optional custom ID (15 chars)
 *         name:
 *           type: string
 *         description:
 *           type: string
 *         lat:
 *           type: number
 *           minimum: -90
 *           maximum: 90
 *         lon:
 *           type: number
 *           minimum: -180
 *           maximum: 180
 *         distance_from_start:
 *           type: number
 *         icon:
 *           type: string
 *         author:
 *           type: string
 *           description: Author user ID (15 chars)
 *         photos:
 *           type: array
 *           items:
 *             type: string
 *           default: []
 *         trail:
 *           type: string
 *
 *     WaypointUpdateInput:
 *       type: object
 *       properties:
 *         name:
 *           type: string
 *         description:
 *           type: string
 *         lat:
 *           type: number
 *           minimum: -90
 *           maximum: 90
 *         lon:
 *           type: number
 *           minimum: -180
 *           maximum: 180
 *         distance_from_start:
 *           type: number
 *         icon:
 *           type: string
 *         photos:
 *           type: array
 *           items:
 *             type: string
 *         'photos-':
 *           type: string
 *           description: Remove photo by name
 *         'photos+':
 *           type: string
 *           description: Add photo by name
 *
 *     List:
 *       type: object
 *       required:
 *         - id
 *         - name
 *         - author
 *         - public
 *       properties:
 *         id:
 *           type: string
 *           description: List ID (15 chars)
 *         name:
 *           type: string
 *         public:
 *           type: boolean
 *         description:
 *           type: string
 *         author:
 *           type: string
 *           description: Author user ID
 *         trails:
 *           type: array
 *           items:
 *             type: string
 *             description: Trail IDs (15 chars)
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     ListInput:
 *       type: object
 *       required:
 *         - name
 *         - public
 *         - trails
 *         - author
 *       properties:
 *         id:
 *           type: string
 *           description: Optional custom ID (15 chars)
 *         name:
 *           type: string
 *           minLength: 1
 *         public:
 *           type: boolean
 *         description:
 *           type: string
 *         trails:
 *           type: array
 *           items:
 *             type: string
 *             description: Trail ID (15 chars)
 *         author:
 *           type: string
 *
 *     ListUpdateInput:
 *       type: object
 *       properties:
 *         name:
 *           type: string
 *         public:
 *           type: boolean
 *         description:
 *           type: string
 *         trails:
 *           type: array
 *           items:
 *             type: string
 *         'trails-':
 *           type: string
 *           description: Remove trail by ID
 *         'trails+':
 *           type: string
 *           description: Add trail by ID
 *
 *     TrailShare:
 *       type: object
 *       required:
 *         - id
 *         - actor
 *         - trail
 *         - permission
 *       properties:
 *         id:
 *           type: string
 *           description: Share ID (15 chars)
 *         actor:
 *           type: string
 *           format: uri
 *           description: ActivityPub actor IRI
 *         trail:
 *           type: string
 *           description: Trail ID (15 chars)
 *         permission:
 *           type: string
 *           enum: [view, edit]
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     TrailShareInput:
 *       type: object
 *       required:
 *         - actor
 *         - trail
 *         - permission
 *       properties:
 *         actor:
 *           type: string
 *           format: uri
 *           description: ActivityPub actor IRI
 *         trail:
 *           type: string
 *           description: Trail ID (15 chars)
 *         permission:
 *           type: string
 *           enum: [view, edit]
 *
 *     TrailShareUpdateInput:
 *       type: object
 *       required:
 *         - permission
 *       properties:
 *         permission:
 *           type: string
 *           enum: [view, edit]
 *
 *     ListShare:
 *       type: object
 *       required:
 *         - id
 *         - actor
 *         - list
 *         - permission
 *       properties:
 *         id:
 *           type: string
 *           description: Share ID (15 chars)
 *         actor:
 *           type: string
 *           format: uri
 *           description: ActivityPub actor IRI
 *         list:
 *           type: string
 *           description: List ID (15 chars)
 *         permission:
 *           type: string
 *           enum: [view, edit]
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     ListShareInput:
 *       type: object
 *       required:
 *         - actor
 *         - list
 *         - permission
 *       properties:
 *         actor:
 *           type: string
 *           format: uri
 *           description: ActivityPub actor IRI
 *         list:
 *           type: string
 *           description: List ID (15 chars)
 *         permission:
 *           type: string
 *           enum: [view, edit]
 *
 *     ListShareUpdateInput:
 *       type: object
 *       required:
 *         - permission
 *       properties:
 *         permission:
 *           type: string
 *           enum: [view, edit]
 *
 *     TrailLinkShare:
 *       type: object
 *       required:
 *         - id
 *         - trail
 *         - permission
 *       properties:
 *         id:
 *           type: string
 *           description: Share ID (15 chars)
 *         trail:
 *           type: string
 *           description: Trail ID (15 chars)
 *         permission:
 *           type: string
 *           enum: [view, edit]
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     TrailLinkShareInput:
 *       type: object
 *       required:
 *         - trail
 *         - permission
 *       properties:
 *         trail:
 *           type: string
 *           description: Trail ID (15 chars)
 *         permission:
 *           type: string
 *           enum: [view, edit]
 *
 *     TrailLinkShareUpdateInput:
 *       type: object
 *       required:
 *         - permission
 *       properties:
 *         permission:
 *           type: string
 *           enum: [view, edit]
 *
 *     TrailLike:
 *       type: object
 *       required:
 *         - id
 *         - actor
 *         - trail
 *       properties:
 *         id:
 *           type: string
 *           description: Like ID (15 chars)
 *         actor:
 *           type: string
 *           description: User ID (15 chars)
 *         trail:
 *           type: string
 *           description: Trail ID (15 chars)
 *         created:
 *           type: string
 *           format: date-time
 *
 *     TrailLikeInput:
 *       type: object
 *       required:
 *         - actor
 *         - trail
 *       properties:
 *         actor:
 *           type: string
 *           description: User ID (15 chars)
 *         trail:
 *           type: string
 *           description: Trail ID (15 chars)
 *
 *     Follow:
 *       type: object
 *       required:
 *         - id
 *         - follower
 *         - followee
 *       properties:
 *         id:
 *           type: string
 *           description: Follow ID (15 chars)
 *         follower:
 *           type: string
 *           description: Follower user ID
 *         followee:
 *           type: string
 *           description: Followee user ID (15 chars)
 *         created:
 *           type: string
 *           format: date-time
 *
 *     FollowInput:
 *       type: object
 *       required:
 *         - followee
 *       properties:
 *         followee:
 *           type: string
 *           description: Followee user ID (15 chars)
 *
 *     Integration:
 *       type: object
 *       required:
 *         - id
 *         - user
 *       properties:
 *         id:
 *           type: string
 *           description: Integration ID (15 chars)
 *         user:
 *           type: string
 *           description: User ID (15 chars)
 *         strava:
 *           type: object
 *           properties:
 *             clientId:
 *               type: integer
 *             clientSecret:
 *               type: string
 *             routes:
 *               type: boolean
 *             activities:
 *               type: boolean
 *             active:
 *               type: boolean
 *             after:
 *               type: string
 *               format: date
 *             privacy:
 *               type: string
 *               enum: [original, settings]
 *         komoot:
 *           type: object
 *           properties:
 *             email:
 *               type: string
 *               format: email
 *             password:
 *               type: string
 *             completed:
 *               type: boolean
 *             planned:
 *               type: boolean
 *             active:
 *               type: boolean
 *             privacy:
 *               type: string
 *               enum: [original, settings]
 *         hammerhead:
 *           type: object
 *           properties:
 *             email:
 *               type: string
 *               format: email
 *             password:
 *               type: string
 *             completed:
 *               type: boolean
 *             planned:
 *               type: boolean
 *             active:
 *               type: boolean
 *             after:
 *               type: string
 *               format: date
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     IntegrationInput:
 *       type: object
 *       required:
 *         - user
 *       properties:
 *         user:
 *           type: string
 *           description: User ID (15 chars)
 *         strava:
 *           type: object
 *         komoot:
 *           type: object
 *         hammerhead:
 *           type: object
 *
 *     IntegrationUpdateInput:
 *       type: object
 *       properties:
 *         strava:
 *           type: object
 *           nullable: true
 *         komoot:
 *           type: object
 *           nullable: true
 *         hammerhead:
 *           type: object
 *           nullable: true
 *
 *     Notification:
 *       type: object
 *       required:
 *         - id
 *         - user
 *         - type
 *         - seen
 *       properties:
 *         id:
 *           type: string
 *           description: Notification ID (15 chars)
 *         user:
 *           type: string
 *           description: User ID (15 chars)
 *         type:
 *           type: string
 *           description: Notification type
 *         seen:
 *           type: boolean
 *           default: false
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     NotificationUpdateInput:
 *       type: object
 *       required:
 *         - seen
 *       properties:
 *         seen:
 *           type: boolean
 *           const: true
 *           description: Mark as read
 *
 *     Settings:
 *       type: object
 *       required:
 *         - id
 *       properties:
 *         id:
 *           type: string
 *           description: Settings ID (15 chars)
 *         unit:
 *           type: string
 *           enum: [metric, imperial]
 *         language:
 *           type: string
 *           description: Language code
 *         bio:
 *           type: string
 *           nullable: true
 *         mapFocus:
 *           type: string
 *           enum: [trails, location]
 *         location:
 *           type: object
 *           properties:
 *             name:
 *               type: string
 *             lat:
 *               type: number
 *             lon:
 *               type: number
 *           nullable: true
 *         category:
 *           type: string
 *         tilesets:
 *           type: array
 *           items:
 *             type: object
 *             properties:
 *               name:
 *                 type: string
 *               url:
 *                 type: string
 *                 format: uri
 *           nullable: true
 *         terrain:
 *           type: object
 *           properties:
 *             terrain:
 *               type: string
 *               format: uri
 *             hillshading:
 *               type: string
 *               format: uri
 *           nullable: true
 *         user:
 *           type: string
 *         privacy:
 *           type: object
 *           properties:
 *             account:
 *               type: string
 *               enum: [public, private]
 *             trails:
 *               type: string
 *               enum: [public, private]
 *             lists:
 *               type: string
 *               enum: [public, private]
 *           nullable: true
 *         created:
 *           type: string
 *           format: date-time
 *         updated:
 *           type: string
 *           format: date-time
 *
 *     SettingsInput:
 *       type: object
 *       properties:
 *         unit:
 *           type: string
 *           enum: [metric, imperial]
 *         language:
 *           type: string
 *         bio:
 *           type: string
 *           nullable: true
 *         mapFocus:
 *           type: string
 *           enum: [trails, location]
 *         location:
 *           type: object
 *           properties:
 *             name:
 *               type: string
 *             lat:
 *               type: number
 *             lon:
 *               type: number
 *           nullable: true
 *         category:
 *           type: string
 *         tilesets:
 *           type: array
 *           items:
 *             type: object
 *             properties:
 *               name:
 *                 type: string
 *               url:
 *                 type: string
 *                 format: uri
 *           nullable: true
 *         terrain:
 *           type: object
 *           properties:
 *             terrain:
 *               type: string
 *               format: uri
 *             hillshading:
 *               type: string
 *               format: uri
 *           nullable: true
 *         user:
 *           type: string
 *         privacy:
 *           type: object
 *           properties:
 *             account:
 *               type: string
 *               enum: [public, private]
 *             trails:
 *               type: string
 *               enum: [public, private]
 *             lists:
 *               type: string
 *               enum: [public, private]
 *           nullable: true
 *
 *     ListResult:
 *       type: object
 *       properties:
 *         page:
 *           type: integer
 *         perPage:
 *           type: integer
 *         totalItems:
 *           type: integer
 *         totalPages:
 *           type: integer
 *         items:
 *           type: array
 *           items:
 *             type: object
 *
 *     Error:
 *       type: object
 *       required:
 *         - message
 *       properties:
 *         message:
 *           type: string
 *         code:
 *           type: integer
 *         data:
 *           type: object
 */
export {};
