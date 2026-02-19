#!/bin/ash

# API endpoint URL
API_URL="http://localhost:3000/api/v1"

# Folder containing files to upload
UPLOAD_FOLDER=$UPLOAD_FOLDER

# Credentials for login
USERNAME=$UPLOAD_USER
PASSWORD=$UPLOAD_PASSWORD

login() {
    local username="$1"
    local password="$2"

    response=$(/curl -c cookie.txt --location --request POST "$API_URL/auth/login" --header 'Content-Type: application/json' --data-raw "{\"username\": \"$username\", \"password\": \"$password\"}")

    # Check if login was successful (look for "200 OK" in response headers)
    if [ $? -eq 0 ] && [ "$(echo "$response" | grep -c "token")" -eq 1 ]; then
        echo "[INFO] [$(date +"%T")]: Login successful. Cookie obtained." > /proc/1/fd/1
    else
        echo "[ERROR] [$(date +"%T")]: Login failed. Unable to obtain cookie." > /proc/1/fd/1
        exit 1
    fi
}

# Function to upload file and delete if successful
upload_and_delete() {
    local file="$1"

    ls "$file"
    
    # API call to upload file
    response=$(/curl -b cookie.txt --location --request PUT "$API_URL/trail/upload" --header 'Content-Type: multipart/form-data' -F "file=@-" -F "name=$base_name" <"$file")
    # Check if API call was successful (status code 200)
    if [ $? -eq 0 ] && [ "$(echo "$response" | grep -c "author")" -eq 1 ]; then
        echo "[INFO] [$(date +"%T")]: File $file uploaded successfully." > /proc/1/fd/1
        # Delete the file
        rm "$file"
        echo "[INFO] [$(date +"%T")]: File $file deleted."
    else
        echo $response
        echo "[ERROR] [$(date +"%T")]: Failed to upload file $file." > /proc/1/fd/1
    fi
}
# Login to obtain cookie
if [ -n "$USERNAME" ] && [ -n "$PASSWORD" ]; then
    echo "[INFO] [$(date +"%T")]: Starting auto-upload" > /proc/1/fd/1

    login "$USERNAME" "$PASSWORD"
    # Iterate over each file in the folder
    for file in "$UPLOAD_FOLDER"/*; do
        # Check if file exists and is a regular file
        if [ -f "$file" ]; then
            upload_and_delete "$file"
        fi
    done

    echo "[INFO] [$(date +"%T")]: Auto-upload completed" > /proc/1/fd/1

fi
