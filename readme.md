# Sugoi
Personal single-user self-hosted web-based manga reader.

# Disclaimer
This is a preview release not intended for general public usage.

The setup for this project is guaranteed to be user-unfriendly.

# Getting started

## Install Go 1.21+
    https://go.dev/dl/
    
## Grab the latest version of Sugoi
    https://github.com/sugoi-reader/sugoi-reader/archive/refs/heads/master.zip

## Extract it somewhere and open the root folder on terminal
run `go build`. Go should download dependencies and run everything automatically.

(If it doesn't work, probably your Go environment isn't set correctly. Git gud.)

## Copy config/sugoi.sample.json to config/sugoi.json

Settings available:
| Setting             | Value                | Description                                                                                      
| ------------------- | -------------------- | -------------------------------------------------------------------------------------------------
| Debug               | true|false           | includes additional debug info on console when running. Change to false on production environment
| CacheThumbnails     | true|false           | Set to false to disable storage of thumbnails (it will run slow af tho)                          
| CacheDir            | "./cache/"           | Path to the folder where thumbnails will be stored
| DatabaseDir         | "./db/"              | Path to the folder where the search database will be stored
| ServerHost          | "127.0.0.1"          | Server host. change to 0.0.0.0 or something else to allow external access within your network
| ServerPort          | 80                   | Server port. Duh.
| SessionCookieName   | "sugoi"              | Cookies shit
| SessionCookieMaxAge | 3600                 | Cookies shit
| SessionCookieKey    | ""                   | Used to encode cookies. Change this to a random 64 byte string
| DirVars             | {}                   | See below
| Users               | {}                   | See below

## Set up your DirVars
DirVars is a list of templates to be replaced in your db/files.txt file. Each entry is supposed to point to the root of a repository of galleries.

## Set up user access
This just prevents unauthorized access within your network. Scores and marks are shared between all users.
Use the -h parameter to hash a new password and store it in your sugoi.json file.

## Set your db/files.txt file
Format:
```
{{MYREPO}}/SadPanda/something.zip
{{MYREPO}}/SadPanda/other thing.zip
{{MYREPO}}/HappyPanda/not so happy.cbz
```
Each line is a gallery. {{MYREPO}} refers to templates set in DirVars.

## Run the compiled file
If everything is ok, you'll get "uwu".
You should be able to access http://localhost and login with "user"/"password".
Go to http://localhost/system and click on "Reload database". It should list all galleries in your files.txt.
Go back and click on "Reindex database". Go to "Status" and press F5 until you get a "100% done!" message.

## Good luck