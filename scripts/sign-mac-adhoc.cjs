const path = require('node:path');
const { signAsync } = require('@electron/osx-sign');

// CI has no Developer ID certificate. Seal the completed bundle before creating
// DMG/ZIP packages instead of leaving Electron's executable-only signature.
// Use afterPack so this also runs on pull requests, where builder skips signing.
module.exports = async function signMacAdhoc(context) {
  if (context.electronPlatformName !== 'darwin') return;

  const app = path.join(
    context.appOutDir,
    `${context.packager.appInfo.productFilename}.app`,
  );
  console.log(`Ad-hoc signing ${app}`);
  await signAsync({
    app,
    platform: 'darwin',
    identity: '-',
    identityValidation: false,
    preAutoEntitlements: false,
    preEmbedProvisioningProfile: false,
    // An ad-hoc identity has no Team ID for hardened library validation.
    optionsForFile: () => ({ hardenedRuntime: false, timestamp: 'none' }),
  });
};
