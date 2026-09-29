# envcheck

A small Go CLI for comparing production .env files with .env_sample files.

## Features

- Compare .env with .env_sample.
- Report missing keys.
- Report keys that exist but are empty/unconfigured.
- Interactive choice to copy missing keys.
- -c copies missing keys automatically.
- -n performs a dry run.
- -r scans immediate project subdirectories.
- Preserve .env_sample key order.
- Copy relevant comments from the sample file.
- Never copy sample values.
- Mark newly added keys as requiring configuration.
- Do not duplicate keys on reruns.
- Create a backup before modifying .env.
- Custom file names are supported.
- Linux binaries can be distributed through GitHub Releases.

## Installation

Production servers do not need Go or the source repository. Download the appropriate binary from a GitHub Release.

For x86_64:

    envcheck-linux-amd64

For ARM64:

    envcheck-linux-arm64

Put it in a user-owned directory:

    mkdir -p ~/bin
    cp envcheck-linux-amd64 ~/bin/envcheck
    chmod +x ~/bin/envcheck

Add it to PATH:

    echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
    source ~/.bashrc

Verify:

    envcheck -v

No sudo and no install.sh are required.

## Usage

    envcheck [options]

Options:

    -d DIR    Directory to check; default: .
    -e FILE   Environment file; default: .env
    -s FILE   Sample file; default: .env_sample
    -c        Copy missing keys automatically
    -r        Scan immediate subdirectories
    -n        Dry run; never modify files
    -v        Show version
    -h        Show help

## Examples

Check the current directory:

    envcheck

Check one application:

    envcheck -d /opt/deployment/myapp

Dry run:

    envcheck -d /opt/deployment/myapp -n

Scan applications one level below deployment:

    envcheck -d /opt/deployment -r

Automatically add missing keys:

    envcheck -d /opt/deployment -r -c

Use custom file names:

    envcheck -d /opt/deployment/myapp -e .env.production -s .env.example

## Interactive Mode

Without -c, missing keys are listed and the user is asked whether to copy them:

    [c] Copy missing keys
    [i] Ignore

Only completely missing keys are copied.

Existing empty keys are reported as unconfigured and are not copied again.

## Example

Given .env_sample:

    # Database username
    DB_USER=sample-user

    # Database password
    DB_PASSWORD=sample-password

And .env:

    DB_USER=production-user

The missing entry is added as:

    # Database password
    # Added by envcheck - VALUE REQUIRED
    DB_PASSWORD=

The value from .env_sample is never copied.

## Rerun Behavior

After the previous operation, another run does not add another DB_PASSWORD.

It reports:

    Unconfigured:
      DB_PASSWORD

After the administrator sets:

    DB_PASSWORD=real-production-password

the key is reported as configured.

## Order and Comments

Missing keys are added in the same order they appear in .env_sample.

Comments associated with the sample key are copied before the new key. The generated value is always empty.

## Production Safety

- Existing values are not overwritten.
- Sample values are never copied.
- Missing values are added empty.
- Empty values remain clearly marked as unconfigured.
- Missing keys are not duplicated on reruns.
- Dry run mode does not modify files.
- A backup is created before modification.

## Development

Clone the repository:

    git clone https://github.com/abhilash-sixthsense/envcheck.git
    cd envcheck

Initialize the module only if go.mod does not exist:

    go mod init github.com/abhilash-sixthsense/envcheck
    go mod tidy

The project uses the Go standard library.

During development, use:

    go run .

Examples:

    go run . -h
    go run . -d .
    go run . -d ./testdata/myapp -n
    go run . -d ./testdata -r

Format:

    gofmt -w main.go

Test:

    go test ./...

## Local Builds

Local build output goes into:

    localdist/

This directory is ignored by Git.

Ruld locally:

    ./scripts/build.sh

Production binaries are published through GitHub Releases.

## Release Process

    ./scripts/release.sh v1.0.0

    git push origin v1.0.0

GItHub Actions then builds the release binaries and publishes them to the GitHub Release.

Production should download the binary from the release rather than cloning the source repository.

## Repository Structure

    envcheck/
    main.go
    go.mod
    README.md
    LICENSE
    .gitignore
    scripts/
        build.sh
        test.sh
        release.sh
    .github/workflows/
        ci.yml
        release.yml

The local build output is:

    localdist/

Make sure localdist/ is in .gitignore.

## Supported Platforms

- Linux AMD64
- Linux ARM64

    uname -m

x86_64 -> envcheck-linux-amd64
 aarch64 -> envcheck-linux-arm64

## Checksums

    sha256sum -c checksums.txt

## Versioning

    v1.0.0
    v1.1.0
    v1.1.1

## License

See LICENSE.
