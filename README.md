# PhotoRoller

PhotoRoller imports photos from a camera/SD-card DCIM folder and organizes them into dated album folders.

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
