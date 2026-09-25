export function Nav() {
  return (
    <nav className="sticky top-0 z-20 border-b border-line bg-bg/80 backdrop-blur-md">
      <div className="mx-auto flex h-14 max-w-5xl items-center gap-2.5 px-5">
        <a href="#" className="flex items-center gap-2 text-[14.5px] font-semibold tracking-tight">
          <img src="/dash.svg" alt="" width={20} height={20} className="rounded-[5px]" />
          Dash
        </a>
        <div className="ml-4 hidden gap-0.5 sm:flex">
          {[
            ["Boards", "#renderers"],
            ["Monitors", "#monitors"],
            ["Stack", "#stack"],
            ["Install", "#install"],
          ].map(([label, href]) => (
            <a
              key={href}
              href={href}
              className="rounded-md px-2.5 py-1.5 text-[13px] text-dim transition-colors hover:bg-surface-hover hover:text-text"
            >
              {label}
            </a>
          ))}
          <a
            href="https://github.com/Dvorinka/Dash/tree/main/docs"
            className="rounded-md px-2.5 py-1.5 text-[13px] text-dim transition-colors hover:bg-surface-hover hover:text-text"
          >
            Docs
          </a>
        </div>
        <div className="ml-auto flex items-center gap-2">
          <a
            href="https://github.com/Dvorinka/Dash"
            aria-label="GitHub repository"
            className="flex h-[34px] w-[34px] items-center justify-center rounded-lg border border-line bg-surface text-dim transition-colors hover:border-line-strong hover:text-text"
          >
            <svg width="15" height="15" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
              <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
            </svg>
          </a>
          <a
            href="#install"
            className="flex h-[34px] items-center rounded-lg bg-text px-3.5 text-[13px] font-medium text-bg transition-transform duration-100 active:scale-[0.97]"
          >
            Get started
          </a>
        </div>
      </div>
    </nav>
  );
}
