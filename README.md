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

- Source: `/media/camera/DCIM/`
- Target: `/mnt/data/assets/__albums/`

PhotoRoller uses `rclone`, so make sure it is installed and available in `PATH`.
