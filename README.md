# aircam-snapshot
Small application for providing an unauthenticated snapshot on Ubiquiti AirCams.

## Justification

The AirCam is a deprecated IP camera created by Ubiquiti. It released with a decent feature set initially, allowing for the ability to view the camera either as an RTSP stream or via snapshots directly from the camera itself. However, with the introduction of the AirVision NVR (now referred to as UniFi Video), Ubiquiti began deliberately removing feature availability from independent cameras in favor of providing the same functionality through the NVR. To make matters worse, Ubiquiti has long since deprecated AirCams within their NVR solution, making the camera essentially worthless on newer firmware.

One feature provided in later firmware is the ability to provide an unauthenticated snapshot from the camera. This is very useful for the [sunlapse](https://github.com/adammillerio/sunlapse) utility, which creates timelapses using these still images. Unfortunately, because later versions of the camera firmware are not very useful, I have remained on legacy version v1.1.5, which does not have an authenticated snapshot. In addition, the camera does not use a standard authentication method, or provide an API of any sort.

## Information

This is a simple tool which is used to provide access to unauthenticated snapshots. It does this by manually receiving and authenticating a session cookie, and then keeping this session alive with the camera. It then exposes the same HTTP route `/snapshot.cgi` but proxies the request using the authenticated session. This allows for access to an unauthenticated snapshot on earlier firmware.

## Configuration

This tool has several configuration values, which are detailed below:

| Name  | Default  | Description  |
|---|---|---|
| SNAPSHOT_URL | N/A | URL of the AirCam (e.g. https://192.168.1.5)
| SNAPSHOT_USERNAME | N/A | Username to login to the AirCam |
| SNAPSHOT_PASSWORD | N/A | Password to login to the AirCam |
| SNAPSHOT_IGNORE_SSL | true | Whether or not to ignore self-signed certificates |
| SNAPSHOT_PORT | 8000 | Port for the local HTTP server to listen on |
| SNAPSHOT_KEEPALIVE_PERIOD | 10 | Period in minutes to make keepalive requests to the AirCam |
