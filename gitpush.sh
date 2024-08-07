#!/bin/bash

# Check if a commit message was provided
if [ -z "$1" ]; then
  echo "Usage: $0 <commit-message>"
  exit 1
fi

# Remove surrounding quotes from the commit message if present
commit_message=$(echo "$1" | sed 's/^"\(.*\)"$/\1/')

# Add all changes to the staging area
git add .

# Commit with the provided message
git commit -m "$commit_message"

# Push the changes to the remote repository
git push
