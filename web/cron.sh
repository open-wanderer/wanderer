#!/bin/ash

# API endpoint URL
API_URL="http://localhost:3000/api/v1"

# Credentials for login
USERNAME=$UPLOAD_USER
PASSWORD=$UPLOAD_PASSWORD

# to use this script outside of the running docker container
# please remove the redirect to /proc/1/fd/1
log() {
    echo "[$1] [$(date +"%T")]: $2" > /proc/1/fd/1
}

log_info() {
    log INFO "$@"
}

log_error() {
    log ERROR "$@"
}

login() {
    local username="$1"
    local password="$2"

    response=$(/curl -c ${COOKIE_FILE} --silent --location --request POST "$API_URL/auth/login" --header 'Content-Type: application/json' --data-raw "{\"username\": \"$username\", \"password\": \"$password\"}")

    # Check if login was successful (look for "200 OK" in response headers)
    if [ $? -eq 0 ] && [ "$(echo "$response" | grep -c "token")" -eq 1 ]; then
        log_info "Login successful. Cookie obtained."
    else
        log_error "Login failed. Unable to obtain cookie."
        exit 1
    fi
}

# Function to upload file and delete if successful
upload_and_delete() {
    local afile="$1"

    # use name of file for trackname
    base_name=$(basename "$afile")

    # prevent files from beeing uploaded by parallel running scripts
    file="$afile".$$
    # rename file with process id as suffix and fail silently if file not exists
    mv "$afile" "$file" || return

    log_info "uploading $afile"

    # API call to upload file
    response=$(/curl -b ${COOKIE_FILE} --silent --location --request PUT "$API_URL/trail/upload" --header 'Content-Type: multipart/form-data' -F "file=@-" -F "name=$base_name" <"$file")
    # Check if API call was successful (status code 200)
    if [ $? -eq 0 ] && [ "$(echo "$response" | grep -c "author")" -eq 1 ]; then
        log_info "File $afile uploaded successfully."
        # Delete the file
        rm "$file"
        log_info "File $afile deleted."
    else
        log_info "$response"
        log_error "Failed to upload file $afile."
        mv "$file" ".$afile"
    fi
}

# Login to obtain cookie
if [ -n "$USERNAME" ] && [ -n "$PASSWORD" ]; then
    log_info "Starting auto-upload"

    COOKIE_FILE=".cookie.$$.txt"

    login "$USERNAME" "$PASSWORD"
    # Iterate over each file in the folder
    for file in "$UPLOAD_FOLDER"/*; do
        # Check if file exists and is a regular file
        if [ -f "$file" ]; then
            upload_and_delete "$file"
        fi
    done

    rm -f ${COOKIE_FILE}
    log_info "Auto-upload completed"
fi
