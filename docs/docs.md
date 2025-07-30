# TinyDB: Current Status

what works now?
- **PUT:** Write new data. Master chooses subvolume; writes to 3 replicas in parallel (all must succeed).
- **GET:** Master uses volume metadata to select a healthy replica for reading (redirects for GET, not proxy).
- **DELETE:** Deletes data from all replicas simultaneously (all must succeed).

## How does it work?
- Two server types: `master` (handles metadata and routing) and `volume` (stores files).
- The master is not a proxy; for `PUT`, it instructs volumes to store, and for `GET`, it redirects clients directly to the volume.
- Files are stored using hash-based keys (nginx/GFS-style filesystem structure).
- Master tracks which volumes contain which data using LevelDB.
- LevelDB is a fast key-value store (widely used for similar infra elsewhere).
- No authentication yet.

## Architectural Notes
- Cluster: 1 Master, N Volumes, all communication via HTTP API.
- GET requests are redirected directly to the relevant volume.
- No consensus/partial-failure tolerance yet: all 3 replicas must succeed on PUT/DELETE for now.

## Limitations and Issues
- Both PUT and DELETE require all replicas to succeed. This reduces availability (if one replica fails, the operation fails).  
  - _Potential fix_: implement write quorum (e.g., require only 2 of 3 to acknowledge).
- No authentication, access control, or automated repair.
-



