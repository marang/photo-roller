# PhotoRoller

![PhotoRoller logo](docs/assets/logo.png)

PhotoRoller imports photos from a camera/SD-card DCIM folder and organizes them into dated album folders.
It reads EXIF timestamps and GPS coordinates, resolves coordinates to place names, writes those names into the target path, and groups related photos into their own time-based folder structure.

Example target layout:

```text
/mnt/data/assets/__albums/
└── 2025-05-11_zell_am_see_and_kaprun/
    ├── 2025-05-11_0910-1232_zell_am_see/
    └── 2025-05-11_1815-2040_kaprun/
```

## Usage

1. Preview what would be created:

   ```sh
   go run . plan
   ```

2. Start the interactive import wizard:

   ```sh
   go run . run
   ```

3. Build a local binary if you prefer:

   ```sh
   go build -o photoroller .
   ./photoroller run
   ```

Defaults:

- Source: auto-detects the first mounted camera/SD-card `DCIM` directory, or use `--source`
- Target: `/mnt/data/assets/__albums/`

PhotoRoller uses `rclone`, so make sure it is installed and available in `PATH`.

## Releases

Linux amd64 and arm64 binaries are published on the
[releases page](https://github.com/marang/photo-roller/releases).
Download the archive for your architecture and `checksums.txt`, then verify it
with `sha256sum --check --ignore-missing checksums.txt` before extracting.
The archive includes `VERSION`; `photoroller --version` reports the binary version.
External runtime tools must be installed separately.

To prepare a release, update `VERSION` and add `docs/releases/vVERSION.md` in a
reviewed PR, run vet/tests and the documented runtime checks, then create the
matching `vVERSION` tag on merged main. The tag workflow runs vet/tests, builds
Linux archives with `scripts/release.sh`, and creates a new GitHub release with
checksums. Existing releases and artifacts are never replaced.
