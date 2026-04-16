import type { UserProfile } from "../api";

interface Props {
  profile: UserProfile;
}

function isRedacted(value: string): boolean {
  return value === "[REDACTED]" || value.startsWith("user_") || value.startsWith("fake_");
}

function Val({ v, mono }: { v: string; mono?: boolean }) {
  const r = isRedacted(v);
  const cls = [mono ? "pf-mono" : "", r ? "pf-redacted" : ""].filter(Boolean).join(" ");
  return <span className={cls || undefined}>{r ? "🔒 " + v : v}</span>;
}

export function ProfileCard({ profile }: Props) {
  return (
    <div className="pf-card">
      {/* Left: identity */}
      <div className="pf-identity-section">
        <div className="pf-id-row">
          <div className="pf-avatar">{profile.name.charAt(0)}</div>
          <div>
            <div className="pf-name">{profile.name}</div>
            <div className="pf-contact"><Val v={profile.email} mono /></div>
          </div>
        </div>
        <div className="pf-details">
          <div className="pf-detail">
            <span className="pf-icon">📞</span>
            <Val v={profile.phone} mono />
          </div>
          <div className="pf-detail">
            <span className="pf-icon">📍</span>
            <Val v={profile.address} mono />
          </div>
        </div>
      </div>

      {/* Right: credit card */}
      <div className="cc">
        <div className="cc-chip" />
        <div className="cc-number"><Val v={profile.card.number} /></div>
        <div className="cc-bottom">
          <div className="cc-field">
            <span className="cc-label">VALID THRU</span>
            <Val v={profile.card.expiry} />
          </div>
          <div className="cc-field">
            <span className="cc-label">CVV</span>
            <Val v={profile.card.cvv} />
          </div>
          <div className="cc-cardholder">{profile.name}</div>
        </div>
      </div>
    </div>
  );
}
