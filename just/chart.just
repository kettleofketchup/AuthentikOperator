# Helm chart versioning recipes
# Call as: just chart::version hotfix, just chart::version minor, etc.

CHART_FILE := env_var("PWD") + "/chart/Chart.yaml"

# Bump chart + app version, commit, tag, and push
[group('release')]
version bump:
    #!/usr/bin/env bash
    set -euo pipefail

    current=$(grep '^version:' {{ CHART_FILE }} | awk '{print $2}')
    IFS='.' read -r major minor patch <<< "$current"

    case "{{ bump }}" in
        major)
            major=$((major + 1)); minor=0; patch=0 ;;
        minor)
            minor=$((minor + 1)); patch=0 ;;
        hotfix|patch)
            patch=$((patch + 1)) ;;
        *)
            echo "Usage: just chart::version [major|minor|hotfix]"
            exit 1 ;;
    esac

    new="${major}.${minor}.${patch}"
    echo "Bumping version: ${current} → ${new}"

    sed -i "s/^version:.*/version: ${new}/" {{ CHART_FILE }}
    sed -i "s/^appVersion:.*/appVersion: \"${new}\"/" {{ CHART_FILE }}

    git add {{ CHART_FILE }}
    git commit -m "release: v${new}"
    git tag -a "v${new}" -m "Release v${new}"

    echo "Tagged v${new}. Push with: git push && git push --tags"

# Show current chart and app versions
[group('release')]
current:
    @echo "chart:   $(grep '^version:' {{ CHART_FILE }} | awk '{print $2}')"
    @echo "app:     $(grep '^appVersion:' {{ CHART_FILE }} | awk '{print $2}' | tr -d '\"')"
    @echo "git tag: $(git describe --tags --always 2>/dev/null || echo 'none')"
