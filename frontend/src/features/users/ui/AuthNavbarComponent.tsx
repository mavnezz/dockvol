import { ThemeToggleComponent } from '../../../shared/ui/ThemeToggleComponent';

export function AuthNavbarComponent() {
  return (
    <div className="flex h-[65px] items-center justify-center px-5 pt-5 sm:justify-start">
      <div className="flex items-center gap-3 hover:opacity-80">
        <a href="https://github.com/mavnezz/dockvol" target="_blank" rel="noreferrer">
          <img className="h-[45px] w-[45px] p-1" src="/logo.svg" />
        </a>

        <div className="text-xl font-bold">
          <a
            href="https://github.com/mavnezz/dockvol"
            className="!text-blue-600"
            target="_blank"
            rel="noreferrer"
          >
            DockVol
          </a>
        </div>
      </div>

      <div className="mr-3 ml-auto hidden items-center gap-5 sm:flex">
        <ThemeToggleComponent />
      </div>
    </div>
  );
}
