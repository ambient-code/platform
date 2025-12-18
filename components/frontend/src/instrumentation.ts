/**
 * Next.js instrumentation - runs once on server startup
 * https://nextjs.org/docs/app/building-your-application/optimizing/instrumentation
 */

export function register() {
  if (process.env.NEXT_RUNTIME === 'nodejs') {
    // Log build information on server startup
    console.log('==============================================');
    console.log('Frontend - Build Information');
    console.log('==============================================');
    console.log(`Version:     ${process.env.NEXT_PUBLIC_GIT_VERSION || 'unknown'}`);
    console.log(`Commit:      ${process.env.NEXT_PUBLIC_GIT_COMMIT || 'unknown'}`);
    console.log(`Branch:      ${process.env.NEXT_PUBLIC_GIT_BRANCH || 'unknown'}`);
    console.log(`Repository:  ${process.env.NEXT_PUBLIC_GIT_REPO || 'unknown'}`);
    console.log(`Built:       ${process.env.NEXT_PUBLIC_BUILD_DATE || 'unknown'}`);
    console.log(`Built by:    ${process.env.NEXT_PUBLIC_BUILD_USER || 'unknown'}`);
    console.log('==============================================');
  }
}

