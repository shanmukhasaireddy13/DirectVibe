import { A } from '@solidjs/router';

export function Footer() {
  return (
    <footer class="app-footer">
      <span>&copy; {new Date().getFullYear()} DirectVibe. All rights reserved.</span>
      <A href="/privacy">Privacy Policy</A>
      <A href="/terms">Terms of Use</A>
    </footer>
  );
}
