import Link from "next/link";

export default function LoggedOut() {
  return (
    <div style={{ display: "flex", justifyContent: "center", alignItems: "center", height: "100vh", fontFamily: "system-ui" }}>
      <div style={{ textAlign: "center" }}>
        <h1>Signed out</h1>
        <p>You have been signed out of Ambient Code.</p>
        <Link href="/" style={{ color: "#0066cc" }}>Sign in again</Link>
      </div>
    </div>
  );
}
