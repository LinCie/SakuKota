import { hasErrorCode, writeErrorMessage } from "~/shared/write-error";

/** A message for a failed kecamatan write, naming the conflicts the admin can fix. */
export function subDistrictErrorMessage(error: Error): string {
  if (hasErrorCode(error, "SUB_DISTRICT_EXISTS")) {
    return "Nama atau slug sudah dipakai kecamatan lain.";
  }
  if (hasErrorCode(error, "SUB_DISTRICT_IN_USE")) {
    return "Kecamatan ini masih punya tempat. Pindahkan atau hapus tempatnya dulu.";
  }
  return writeErrorMessage(error);
}
