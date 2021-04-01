# TODO for the support of encrypted cloning/snapshots

## What works now 🥳

1. snapshot (rbd images) have their DEK in the KMS
2. restored from snapshot PVCs have their DEK in the KMS
3. removed VolumeSnapshotContents delete their DEK from the KMS
4. removed restored/cloned PVCs delete their DEK from the KMS

## What doesn't work yet

1. encrypted PVCs have type "crypt_LUKS", not "crypt"?!

## What to validate

- are the DEKs cleaned up correctly, no additional (temporary?) DEKs stored
- run a pod on a cloned-pvc with pre-poulated contents
- restore with different storage classes, key placed in right location?
