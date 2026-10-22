import { Toast } from "@base-ui/react/toast";

/** The app's toast queue; call `toastManager.add({ title })` from anywhere. */
export const toastManager = Toast.createToastManager();
