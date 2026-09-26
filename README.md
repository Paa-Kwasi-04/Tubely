# Tubely

Tubely is a **learning project built while following Boot.dev's "Learn File Servers and CDNs with S3 and CloudFront" course**.

The project is strictly for learning. It explores how a Go backend can handle authentication, video metadata, file uploads, local storage, video processing, Amazon S3 object storage, and CDN-style delivery.

> This is not intended to be a production-ready video platform. The code follows a guided course whose scaffolding and hints are progressively reduced as the concepts become more advanced.

## What I'm Learning

This project brings several backend and cloud concepts together:

- Go HTTP servers with net/http
- REST-style API design
- JSON request/response handling
- User registration and authentication
- Argon2id password hashing
- JWT access tokens
- Refresh tokens and revocation
- SQLite persistence
- Multipart file uploads
- Local filesystem storage
- Video processing with FFmpeg
- Video metadata inspection with FFprobe
- Amazon S3 object storage
- AWS SDK for Go v2
- CDN/CloudFront concepts
- UUID-based resources
- Environment-based configuration
- HTTP cache-control middleware

## Current Functionality

### Users

Tubely supports:

- User registration
- Argon2id password hashing
- Login
- JWT access tokens
- Refresh tokens
- Refresh-token revocation

### Videos

Authenticated users can:

- Create video metadata
- Retrieve their videos
- Retrieve a video by ID
- Delete their own videos
- Upload video files
- Upload thumbnails

Video metadata includes:

- Title
- Description
- Owner
- Thumbnail URL
- Video URL
- Creation/update timestamps

### Video Upload Pipeline

The current video upload flow is:

~~~text
Client
  │
  │ multipart/form-data
  ▼
Go HTTP server
  │
  ├── Validate JWT
  ├── Check video ownership
  ├── Validate MP4 content type
  ├── Save upload temporarily
  ├── Process with FFmpeg
  ├── Inspect aspect ratio with FFprobe
  ├── Generate random object key
  ├── Upload processed video to S3
  └── Store delivery URL in SQLite
~~~

Videos are currently classified as:

- landscape — 16:9
- portrait — 9:16
- other — other aspect ratios

The processed video is uploaded to the configured S3 bucket using a randomly generated object key.

### Thumbnail Uploads

Thumbnails are currently stored on the local filesystem rather than S3.

Supported types:

- JPEG
- PNG

A random filename is generated and the resulting asset is served through the local /assets/ route.

## API

### Authentication

#### Create a user

~~~http
POST /api/users
Content-Type: application/json
~~~

Example:

~~~json
{
  "email": "user@example.com",
  "password": "password123"
}
~~~

#### Login

~~~http
POST /api/login
Content-Type: application/json
~~~

A successful login returns the user, a JWT access token, and a refresh token.

The current implementation creates access tokens with a **30-day lifetime** and refresh tokens with a **60-day lifetime**.

#### Refresh an access token

~~~http
POST /api/refresh
Authorization: Bearer <refresh-token>
~~~

#### Revoke a refresh token

~~~http
POST /api/revoke
Authorization: Bearer <refresh-token>
~~~

### Video Metadata

#### Create video metadata

~~~http
POST /api/videos
Authorization: Bearer <access-token>
Content-Type: application/json
~~~

Example:

~~~json
{
  "title": "My Video",
  "description": "A test video"
}
~~~

#### Get the authenticated user's videos

~~~http
GET /api/videos
Authorization: Bearer <access-token>
~~~

Videos are returned newest-first.

#### Get a video

~~~http
GET /api/videos/{videoID}
~~~

#### Delete a video

~~~http
DELETE /api/videos/{videoID}
Authorization: Bearer <access-token>
~~~

The authenticated user must own the video.

### File Uploads

#### Upload a video

~~~http
POST /api/video_upload/{videoID}
Authorization: Bearer <access-token>
Content-Type: multipart/form-data
~~~

The multipart field is:

~~~text
video
~~~

The request body is limited to 1 GB and the current implementation accepts MP4 uploads.

#### Upload a thumbnail

~~~http
POST /api/thumbnail_upload/{videoID}
Authorization: Bearer <access-token>
Content-Type: multipart/form-data
~~~

The multipart field is:

~~~text
thumbnail
~~~

Supported types are JPEG and PNG.

### Development Reset

~~~http
POST /admin/reset
~~~

The endpoint only works when:

~~~env
PLATFORM=dev
~~~

It clears the development database.

> Warning: this is a destructive development endpoint.

## Storage Architecture

Tubely currently uses three storage/delivery concepts.

### SQLite

SQLite stores application metadata:

~~~text
users
refresh_tokens
videos
~~~

The database is automatically initialized when the application starts.

### Local Filesystem

Local storage is currently used for:

- Thumbnails
- Static application assets

Configured with:

~~~env
FILEPATH_ROOT="./app"
ASSETS_ROOT="./assets"
~~~

### Amazon S3

Processed video files are uploaded to the configured S3 bucket.

Relevant configuration:

~~~env
S3_BUCKET="your-bucket-name"
S3_REGION="us-east-2"
S3_CF_DISTRO="your-cdn-base-url"
~~~

The application uses the AWS SDK for Go v2 and loads AWS credentials through the standard AWS configuration.

For example:

~~~bash
aws configure
~~~

## CDN / CloudFront Concept

The course explores the relationship between:

~~~text
Application
    │
    ▼
S3 object storage
    │
    ▼
CloudFront / CDN
    │
    ▼
Client
~~~

The current application stores a delivery URL using the configured S3_CF_DISTRO value and generated S3 object key.

The repository should therefore be viewed as a **learning implementation of the storage/CDN architecture**, not as a complete production deployment.

## Environment Variables

Copy the example configuration:

~~~bash
cp .env.example .env
~~~

Current configuration:

~~~env
DB_PATH="./tubely.db"
JWT_SECRET="replace-with-a-secret"
PLATFORM="dev"
FILEPATH_ROOT="./app"
ASSETS_ROOT="./assets"
S3_BUCKET="your-s3-bucket"
S3_REGION="us-east-2"
S3_CF_DISTRO="your-cdn-base-url"
PORT="8091"
~~~

| Variable | Purpose |
| --- | --- |
| DB_PATH | Path to the SQLite database |
| JWT_SECRET | Secret used to sign JWT access tokens |
| PLATFORM | Application environment; dev enables reset |
| FILEPATH_ROOT | Directory served under /app/ |
| ASSETS_ROOT | Directory used for uploaded thumbnails |
| S3_BUCKET | S3 bucket for processed videos |
| S3_REGION | AWS region containing the bucket |
| S3_CF_DISTRO | Base URL used for video delivery URLs |
| PORT | HTTP server port |

**Do not commit real secrets, .env files, or AWS credentials.**

## Requirements

You will need:

- Go 1.27.1 or later
- FFmpeg
- FFprobe
- AWS CLI
- AWS configuration/account access for the S3 portions of the course

The Go module currently uses:

- AWS SDK for Go v2
- SQLite
- Argon2id
- JWT
- UUID
- godotenv

## Installation

### 1. Clone the repository

~~~bash
git clone https://github.com/Paa-Kwasi-04/Tubely.git
cd Tubely
~~~

### 2. Download dependencies

~~~bash
go mod download
~~~

### 3. Install FFmpeg

FFmpeg and FFprobe must both be available in PATH.

Ubuntu/Debian:

~~~bash
sudo apt update
sudo apt install ffmpeg
~~~

macOS:

~~~bash
brew update
brew install ffmpeg
~~~

Verify:

~~~bash
ffmpeg -version
ffprobe -version
~~~

### 4. Configure AWS

Install the AWS CLI and configure credentials if you are working with S3:

~~~bash
aws configure
~~~

The AWS SDK used by Tubely loads credentials through the standard AWS configuration.

### 5. Configure the environment

~~~bash
cp .env.example .env
~~~

Update the values in .env for your local environment and AWS resources.

## Running the Server

Start the application with:

~~~bash
go run .
~~~

With the example configuration, the application listens on:

~~~text
http://localhost:8091/app/
~~~

On startup, the application:

1. Loads .env.
2. Opens the SQLite database.
3. Creates the required tables if they do not exist.
4. Loads AWS configuration.
5. Creates the S3 client.
6. Ensures the assets directory exists.
7. Registers the HTTP routes.
8. Starts the HTTP server.

## Sample Files

The repository includes a script for downloading the sample media used by the course:

~~~bash
./samplesdownload.sh
~~~

This creates a samples/ directory containing sample images and videos.

## Project Structure

~~~text
Tubely/
├── app/                         # Static application files
├── internal/
│   ├── auth/
│   │   └── auth.go              # JWT, hashing, token parsing
│   └── database/
│       ├── database.go          # SQLite client, schema, reset
│       ├── users.go             # User operations
│       ├── refresh_tokens.go    # Refresh-token operations
│       └── videos.go            # Video metadata operations
├── assets.go                    # Ensures assets directory exists
├── cache.go                     # No-cache middleware
├── handler_login.go             # Login
├── handler_refresh.go           # Refresh/revoke
├── handler_users.go             # User creation
├── handler_video_meta.go        # Video metadata operations
├── handler_upload_video.go      # Video upload + S3
├── handler_upload_thumbnail.go  # Thumbnail upload
├── videoProcessing.go            # FFmpeg/FFprobe operations
├── json.go                      # JSON response helpers
├── reset.go                     # Development reset
├── main.go                      # Configuration and server startup
├── samplesdownload.sh           # Course sample-media downloader
├── .env.example                 # Environment template
├── go.mod
└── go.sum
~~~

## Architecture

At a high level, the video upload path is:

~~~text
                    Client
                       │
                       ▼
                  HTTP Server
                       │
                       ▼
                Upload Handler
                       │
             ┌─────────┴─────────┐
             │                   │
             ▼                   ▼
        JWT validation       SQLite metadata
             │
             ▼
        Temporary file
             │
             ▼
          FFmpeg
       fast-start MP4
             │
             ▼
         FFprobe
       aspect ratio
             │
             ▼
       Random S3 key
             │
             ▼
        Amazon S3
             │
             ▼
     CDN-style delivery URL
~~~

Thumbnail uploads currently use:

~~~text
Client
  │
  ▼
Thumbnail handler
  │
  ├── JWT validation
  ├── Ownership check
  ├── Validate JPEG/PNG
  └── Save to local assets directory
              │
              ▼
          /assets/...
~~~

## Authentication Flow

### Registration

~~~text
Email + password
       │
       ▼
Argon2id
       │
       ▼
SQLite users table
~~~

### Login

~~~text
Email + password
       │
       ▼
Find user
       │
       ▼
Verify password hash
       │
       ├───────────────┐
       ▼               ▼
   JWT access      Refresh token
       │               │
       │               ▼
       │          SQLite storage
       ▼
     Client
~~~

JWT access tokens use the issuer:

~~~text
tubely-access
~~~

The JWT subject contains the user's UUID.

## Learning Scope

This repository is deliberately **not presented as production software**.

The goal is to understand concepts introduced throughout the course, including:

- Why large media files can be separated from application metadata
- How object storage such as S3 can store media
- How a CDN can sit in front of object storage
- How a backend stores metadata separately from media
- How multipart uploads work
- How temporary files can be used during processing
- How FFmpeg can prepare video files for web delivery
- How FFprobe can inspect media metadata
- How authentication fits into upload APIs
- How AWS services can be integrated into a Go application

Some parts are intentionally simplified or still evolving as the course progresses.

## Testing

There are currently no dedicated automated tests documented for the application.

When adding tests, use:

~~~bash
go test ./...
~~~

## Course Attribution

This repository is based on the starter project and guided curriculum from Boot.dev's:

**Learn File Servers and CDNs with S3 and CloudFront**

Boot.dev:

https://www.boot.dev/

Course:

https://www.boot.dev/courses/learn-file-servers-s3-cloudfront-golang

This repository is maintained as a personal learning project while progressing through the course.

## License

This repository does not currently specify a separate open-source license.
