/**
 * Google Picker API and Google Identity Services (GIS) loader.
 *
 * The Picker API is loaded via a <script> tag (no npm package available).
 * GIS is loaded similarly. This module provides helpers to load both
 * scripts and ensure they are ready before use.
 */

const PICKER_API_URL = "https://apis.google.com/js/api.js";
const GIS_URL = "https://accounts.google.com/gsi/client";

let pickerApiLoaded = false;
let gisLoaded = false;

function loadScript(src: string): Promise<void> {
  return new Promise((resolve, reject) => {
    if (document.querySelector(`script[src="${src}"]`)) {
      resolve();
      return;
    }

    const script = document.createElement("script");
    script.src = src;
    script.async = true;
    script.defer = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error(`Failed to load script: ${src}`));
    document.head.appendChild(script);
  });
}

/**
 * Load the Google API client library and the Picker API.
 * Must be called before using google.picker.
 */
export async function loadPickerApi(): Promise<void> {
  if (pickerApiLoaded) return;

  await loadScript(PICKER_API_URL);

  return new Promise((resolve, reject) => {
    window.gapi.load("picker", {
      callback: () => {
        pickerApiLoaded = true;
        resolve();
      },
      onerror: () => reject(new Error("Failed to load Google Picker API")),
    });
  });
}

/**
 * Load the Google Identity Services (GIS) library.
 * Must be called before using google.accounts.oauth2.
 */
export async function loadGis(): Promise<void> {
  if (gisLoaded) return;

  await loadScript(GIS_URL);
  gisLoaded = true;
}

/**
 * Load both the Picker API and GIS in parallel.
 */
export async function loadGoogleApis(): Promise<void> {
  await Promise.all([loadPickerApi(), loadGis()]);
}

// Type augmentations for the global google APIs
declare global {
  interface Window {
    gapi: {
      load: (
        api: string,
        options: { callback: () => void; onerror: () => void }
      ) => void;
    };
    google: {
      accounts: {
        oauth2: {
          initTokenClient: (config: {
            client_id: string;
            scope: string;
            callback: (response: { access_token: string }) => void;
            error_callback?: (error: { type: string }) => void;
          }) => { requestAccessToken: () => void };
        };
      };
      picker: {
        PickerBuilder: new () => GooglePickerBuilder;
        ViewId: {
          DOCS: string;
          DOCS_IMAGES: string;
          SPREADSHEETS: string;
          FOLDERS: string;
        };
        Feature: {
          MULTISELECT_ENABLED: string;
          SUPPORT_DRIVES: string;
        };
        Action: {
          PICKED: string;
          CANCEL: string;
        };
        DocsView: new (viewId?: string) => GoogleDocsView;
      };
    };
  }

  interface GooglePickerBuilder {
    setOAuthToken(token: string): GooglePickerBuilder;
    setDeveloperKey(key: string): GooglePickerBuilder;
    setAppId(appId: string): GooglePickerBuilder;
    setCallback(callback: (data: GooglePickerResponse) => void): GooglePickerBuilder;
    addView(view: GoogleDocsView | string): GooglePickerBuilder;
    enableFeature(feature: string): GooglePickerBuilder;
    setOrigin(origin: string): GooglePickerBuilder;
    setTitle(title: string): GooglePickerBuilder;
    build(): GooglePicker;
  }

  interface GoogleDocsView {
    setIncludeFolders(include: boolean): GoogleDocsView;
    setMimeTypes(mimeTypes: string): GoogleDocsView;
    setSelectFolderEnabled(enabled: boolean): GoogleDocsView;
  }

  interface GooglePicker {
    setVisible(visible: boolean): void;
  }

  interface GooglePickerResponse {
    action: string;
    docs: GooglePickerDocument[];
  }

  interface GooglePickerDocument {
    id: string;
    name: string;
    mimeType: string;
    url: string;
    sizeBytes: number;
    lastEditedUtc: number;
    iconUrl: string;
    description: string;
    parentId: string;
    serviceId: string;
    type: string;
    isShared: boolean;
  }
}
