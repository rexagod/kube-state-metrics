#!/bin/bash
set -exuo pipefail

CHANGELOG="CHANGELOG.md"

# Fetch the latest two tags.
# NOTE: This assumes a patch release for an older release is not made after a later minor release (which is the case right now).
# Backports will be handled manually, on a case-by-case basis.
last_two_tags=$(git tag --sort=-creatordate | head -n 2)

# read will return a non-zero exit code when it hits EOF, so we need to disable the exit-on-error option.
set +e
IFS=$'\n' read -d '' -r -a tags <<< "$last_two_tags"
set -e

# Get the commits between the two tags.
commits=$(git log --pretty=format:"%h: %an <%ae>: %s" "${tags[1]}".."${tags[0]}" | grep -i -v -e 'fixup' -e 'merge' -e 'dependabot')

# Update the changelog with the latest release notes.
echo -e "## ${tags[0]} / $(date "+%Y-%m-%d")\n\n$commits\n\n$(cat $CHANGELOG)" > $CHANGELOG
