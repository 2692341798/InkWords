#!/bin/bash
# Need a personal access token with repo scope
if [ -z "$GITHUB_TOKEN" ]; then
  echo "Error: GITHUB_TOKEN environment variable is not set."
  echo "Please set it first: export GITHUB_TOKEN=your_token_here"
  exit 1
fi

REPO="2692341798/InkWords"
API_URL="https://api.github.com/repos/$REPO/labels"

echo "Current labels:"
curl -s -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" $API_URL | jq -r '.[].name'

echo "Deleting unwanted default labels..."
for label in "duplicate" "good first issue" "help wanted" "invalid" "wontfix"; do
  # URL encode the label name
  encoded_label=$(echo -n "$label" | jq -sRr @uri)
  echo "Deleting: $label"
  curl -s -X DELETE -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" "$API_URL/$encoded_label"
done

echo "Ensuring core labels exist and are well-formatted..."
curl -s -X POST -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" \
  -d '{"name":"bug","color":"d73a4a","description":"Something isn'\''t working"}' $API_URL

curl -s -X POST -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" \
  -d '{"name":"enhancement","color":"a2eeef","description":"New feature or request"}' $API_URL

curl -s -X POST -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" \
  -d '{"name":"documentation","color":"0075ca","description":"Improvements or additions to documentation"}' $API_URL

curl -s -X POST -H "Authorization: token $GITHUB_TOKEN" -H "Accept: application/vnd.github.v3+json" \
  -d '{"name":"question","color":"d876e3","description":"Further information is requested"}' $API_URL

echo "Done!"
