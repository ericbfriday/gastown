# macOS Darwin System Commands

## Platform-Specific Notes

**OS**: macOS Darwin 25.2.0 (Apple Silicon)
**Shell**: bash (primary), zsh (available)

## Command Differences from Linux

### GNU Tools via Homebrew

macOS uses BSD versions of common utilities by default. GNU versions are available via Homebrew with `g` prefix:

```bash
# BSD (macOS default)
find . -name "*.sh"
sed 's/old/new/' file.txt

# GNU (via Homebrew, higher priority in PATH)
gfind . -name "*.sh"
gsed 's/old/new/' file.txt
```

**Available GNU Tools**:
- gfind (findutils)
- gsed (gnu-sed)
- gawk (gawk)
- Other coreutils

**PATH Priority**: Homebrew paths are first, so unprefixed versions use GNU tools when installed.

## Essential Commands

### File Operations

```bash
# List files
ls -la                      # Detailed list
ls -lh                      # Human-readable sizes
ls -lt                      # Sort by modification time

# Find files
find . -name "pattern"      # Find by name
find . -type f -name "*.sh" # Find shell scripts
find . -mtime -1            # Modified in last 24 hours

# Search content
grep -r "pattern" .         # Recursive search
grep -i "pattern" file      # Case insensitive
grep -n "pattern" file      # With line numbers

# Copy/Move/Delete
cp -r source dest           # Copy recursively
mv source dest              # Move/rename
rm -rf directory            # Remove recursively (BE CAREFUL)
```

### Text Processing

```bash
# View files
cat file.txt                # Print entire file
head -n 20 file.txt         # First 20 lines
tail -n 20 file.txt         # Last 20 lines
tail -f file.log            # Follow file (live updates)
less file.txt               # Paginated view

# Edit in place (BSD sed)
sed -i '' 's/old/new/g' file.txt  # Note: requires empty string for -i

# JSON processing
jq . file.json              # Pretty print
jq '.field' file.json       # Extract field
jq -r '.[] | .name' file.json  # Extract array field
```

### Process Management

```bash
# View processes
ps aux                      # All processes
ps aux | grep process       # Find specific process
pgrep -f "pattern"          # Find by pattern

# Kill processes
pkill -f "loop.sh"          # Kill by pattern
kill -TERM <pid>            # Graceful shutdown
kill -9 <pid>               # Force kill

# Background jobs
./script.sh &               # Run in background
jobs                        # List background jobs
fg %1                       # Bring job to foreground
```

### Disk & File System

```bash
# Disk usage
df -h                       # Disk free space
du -sh directory            # Directory size
du -sh * | sort -h          # Sort by size

# File permissions
chmod +x script.sh          # Make executable
chmod 644 file.txt          # Read/write owner, read others
chmod 755 script.sh         # Executable script

# Ownership
chown user:group file       # Change owner
chown -R user:group dir     # Recursive
```

### Environment & Path

```bash
# View environment
env                         # All variables
echo $PATH                  # Path variable
echo $HOME                  # Home directory

# Set environment
export VAR=value            # Set variable
export PATH=$PATH:/new/path # Add to path

# Which command
which command               # Find command location
type command                # Command type and location
```

### Network

```bash
# Check connectivity
ping hostname               # Ping host
curl https://example.com    # HTTP request
curl -I https://example.com # HTTP headers only

# DNS
nslookup hostname           # DNS lookup
dig hostname                # Detailed DNS

# Ports
lsof -i :8080               # What's using port 8080
netstat -an | grep LISTEN   # Listening ports
```

### Archive & Compression

```bash
# tar (GNU tar via Homebrew)
tar -czf archive.tar.gz dir # Create compressed archive
tar -xzf archive.tar.gz     # Extract archive
tar -tzf archive.tar.gz     # List contents

# zip
zip -r archive.zip dir      # Create zip
unzip archive.zip           # Extract zip
unzip -l archive.zip        # List contents
```

## Package Management (Homebrew)

```bash
# Install packages
brew install package

# Update
brew update                 # Update Homebrew
brew upgrade                # Upgrade packages
brew upgrade package        # Upgrade specific package

# Search
brew search pattern

# Info
brew info package

# List installed
brew list
```

## Python (via uv)

```bash
# Create virtual environment
uv venv

# Install packages
uv pip install package

# Run with specific Python
uv run python script.py

# List available Pythons
uv python list
```

## Node.js (via Volta)

```bash
# Check version
volta list

# Install version
volta install node@20

# Pin version for project
volta pin node@20
```

## Go

```bash
# Build
go build ./...

# Test
go test ./...

# Install tool
go install package@latest

# Clean cache
go clean -cache
```

## Git

```bash
# Status and info
git status
git log --oneline -10
git branch -a

# Fetch and pull
git fetch origin
git pull --rebase origin main

# Commit and push
git add .
git commit -m "message"
git push origin main

# Branch management
git checkout -b branch-name
git branch -d branch-name
git branch --merged
```

## System Monitoring

```bash
# Activity Monitor (GUI)
open -a "Activity Monitor"

# CPU usage
top                         # Interactive
top -l 1 | head -n 10       # One shot

# Memory
vm_stat                     # Virtual memory stats

# System info
sw_vers                     # OS version
uname -a                    # Kernel info
sysctl -a | grep cpu        # CPU info
```

## File Watching

```bash
# Watch command output
watch -n 5 command          # Run every 5 seconds

# Monitor file changes
tail -f file.log            # Follow log file

# Directory monitoring (fswatch via Homebrew)
fswatch -o directory        # Watch for changes
```

## Differences to Note

### Sed (in-place editing)
```bash
# macOS (BSD sed) - requires empty string
sed -i '' 's/old/new/' file

# Linux (GNU sed)
sed -i 's/old/new/' file
```

### Find (case-insensitive)
```bash
# macOS/BSD
find . -iname "pattern"

# Works on both
find . -name "pattern"
```

### Disk Usage (human readable)
```bash
# macOS
du -h -d 1 .

# Linux
du -h --max-depth=1 .
```

### Process Signals
```bash
# Both support
kill -TERM <pid>
kill -INT <pid>
kill -9 <pid>

# macOS specific signal names
kill -s TERM <pid>
```

## Homebrew Paths

**Installation**: `/opt/homebrew` (Apple Silicon)

**Bins**: `/opt/homebrew/bin`
**Libraries**: `/opt/homebrew/lib`
**Includes**: `/opt/homebrew/include`

**Linked into PATH**: Yes (first priority)

## Useful Aliases

```bash
# Common shortcuts (add to ~/.bash_profile)
alias ll='ls -lh'
alias la='ls -lah'
alias ..='cd ..'
alias ...='cd ../..'
alias grep='grep --color=auto'
```

## Terminal Applications

**Default Terminal**: Terminal.app
**Available**: Ghostty (`/Applications/Ghostty.app/Contents/MacOS`)
**Shell**: bash or zsh

## Shell Completions

**Available for**:
- uv (Python)
- git
- npm/yarn (via Volta)

**Loading**: Via `.bash_profile` or `.zshrc`
