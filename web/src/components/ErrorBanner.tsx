export function ErrorBanner({ children }: { children: string }) {
  return (
    <div
      role="alert"
      className="border-b border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300"
    >
      {children}
    </div>
  );
}
