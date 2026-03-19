# Copier template management recipes
# Call as: just copier::update, just copier::diff, etc.

# Update project from template (pulls latest changes)
[group('copier')]
[confirm("This will update your project from the template. Continue?")]
update:
    @echo "Updating project from template..."
    copier update --trust

# Update without prompts (for CI or scripted use)
[group('copier')]
update-auto:
    @echo "Updating project from template (auto-approve)..."
    copier update --trust --defaults

# Show diff between current project and template
[group('copier')]
diff:
    @echo "Showing diff against template..."
    copier update --trust --pretend --diff

# Re-copy template (useful after major template changes)
[group('copier')]
[confirm("This will re-copy the template, potentially overwriting changes. Continue?")]
recopy:
    @echo "Re-copying template..."
    copier recopy --trust

# Show current template answers
[group('copier')]
answers:
    @echo "Current template answers:"
    @cat .copier-answers.yml 2>/dev/null || echo "No .copier-answers.yml found"
