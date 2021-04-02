# TODO for the support of encrypted cloning/snapshots

## What works now 🥳

1. snapshot (rbd images) have their DEK in the KMS
2. restored from snapshot PVCs have the correct DEK in the KMS
4. removed restored/cloned PVCs delete their DEK from the KMS
5. clone-from-pvc has the right DEK
6. run a pod on a restored-from-snapshot-pvc with pre-populated contents

## What doesn't work yet

1. encrypted PVCs have type "crypt_LUKS", not "crypt"?!
   - (PR#1950)
3. removed VolumeSnapshotContents delete their DEK from the KMS
   snapshot_controller.go:531] Check if VolumeSnapshotContent[snapcontent-e33203c2-e509-4492-8674-090b562c5520] should be deleted.
   - This did work before??

## What to validate

- are the DEKs cleaned up correctly, no additional (temporary?) DEKs stored
- restore with different storage classes, key placed in right location?
