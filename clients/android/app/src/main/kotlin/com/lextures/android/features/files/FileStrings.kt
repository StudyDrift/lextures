package com.lextures.android.features.files

import androidx.compose.runtime.Composable
import com.lextures.android.R
import com.lextures.android.core.i18n.L

@Composable fun fileRootLabel(): String = L.text(R.string.mobile_files_root)

@Composable fun fileEmptyFolderTitle(): String = L.text(R.string.mobile_files_emptyFolder)

@Composable fun fileEmptyFolderHint(): String = L.text(R.string.mobile_files_emptyFolderHint)

@Composable fun fileLoadErrorLabel(): String = L.text(R.string.mobile_files_loadError)

@Composable fun fileCacheSizeLabel(): String = L.text(R.string.mobile_files_cacheSize)

@Composable fun fileClearCacheLabel(): String = L.text(R.string.mobile_files_clearCache)

@Composable fun fileClearCacheTitle(): String = L.text(R.string.mobile_files_clearCacheTitle)

@Composable fun fileClearCacheMessage(): String = L.text(R.string.mobile_files_clearCacheMessage)

@Composable fun fileClearCacheConfirmLabel(): String = L.text(R.string.mobile_files_clearCacheConfirm)

@Composable fun fileFolderLabel(): String = L.text(R.string.mobile_files_folder)

@Composable fun fileSavedLabel(): String = L.text(R.string.mobile_files_saved)

@Composable fun fileDownloadLabel(): String = L.text(R.string.mobile_files_download)

@Composable fun fileOpenInLabel(): String = L.text(R.string.mobile_files_openIn)

@Composable fun fileLoadingLabel(): String = L.text(R.string.mobile_files_loading)

@Composable fun fileDownloadOnlyHint(): String = L.text(R.string.mobile_files_downloadOnlyHint)

@Composable fun filePreviewUnavailableLabel(): String = L.text(R.string.mobile_files_previewUnavailable)

@Composable fun fileOfflineUnavailableLabel(): String = L.text(R.string.mobile_files_offlineUnavailable)

@Composable fun fileDownloadErrorLabel(): String = L.text(R.string.mobile_files_downloadError)

@Composable fun fileOpenErrorLabel(): String = L.text(R.string.mobile_files_openError)
