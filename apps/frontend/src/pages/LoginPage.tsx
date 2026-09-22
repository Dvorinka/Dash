import { useState, type FormEvent } from "react";
import { api } from "@/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { t } from "@/i18n";

/** Login/setup gate shown when auth is enabled and no session exists. */
export function LoginPage({ setup, onDone }: { setup: boolean; onDone: () => void }) {
	const [username, setUsername] = useState("");
	const [password, setPassword] = useState("");
	const [err, setErr] = useState("");
	const [busy, setBusy] = useState(false);

	async function submit(e: FormEvent) {
		e.preventDefault();
		setErr("");
		setBusy(true);
		const body = { username: username.trim(), password };
		const { error, response } = setup
			? await api.POST("/api/auth/setup", { body })
			: await api.POST("/api/auth/login", { body });
		setBusy(false);
		if (error) {
			setErr(response.status === 429 ? t("auth.rateLimited") : t("auth.invalid"));
			return;
		}
		onDone();
	}

	return (
		<div className="flex min-h-screen items-center justify-center bg-bg p-6">
			<form
				onSubmit={(e) => void submit(e)}
				className="w-full max-w-[320px] rounded-xl border border-border bg-surface p-6 shadow-lg"
			>
				<div className="mb-5 flex items-center gap-2.5 text-[15px] font-semibold tracking-tight">
					<svg width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden>
						<rect x="1" y="1" width="7" height="7" rx="1.5" fill="currentColor" />
						<rect x="10" y="1" width="7" height="7" rx="1.5" fill="currentColor" opacity=".45" />
						<rect x="1" y="10" width="7" height="7" rx="1.5" fill="currentColor" opacity=".45" />
						<rect x="10" y="10" width="7" height="7" rx="1.5" fill="currentColor" />
					</svg>
					Dash
				</div>
				<h1 className="mb-1 text-[14px] font-medium">{setup ? t("auth.setupTitle") : t("auth.signInTitle")}</h1>
				<p className="mb-4 text-[12px] text-text-faint">{setup ? t("auth.setupHint") : t("auth.signInHint")}</p>
				<div className="grid gap-3">
					<div className="grid gap-1.5">
						<Label htmlFor="auth-user">{t("auth.username")}</Label>
						<Input
							id="auth-user" autoComplete="username" autoFocus
							value={username} onChange={(e) => setUsername(e.target.value)}
						/>
					</div>
					<div className="grid gap-1.5">
						<Label htmlFor="auth-pass">{t("auth.password")}</Label>
						<Input
							id="auth-pass" type="password"
							autoComplete={setup ? "new-password" : "current-password"}
							value={password} onChange={(e) => setPassword(e.target.value)}
						/>
					</div>
					{err && <p className="text-[12px] text-danger">{err}</p>}
					<Button type="submit" disabled={busy || !username.trim() || password.length < 8} className="mt-1">
						{setup ? t("auth.createAccount") : t("auth.signIn")}
					</Button>
				</div>
			</form>
		</div>
	);
}
