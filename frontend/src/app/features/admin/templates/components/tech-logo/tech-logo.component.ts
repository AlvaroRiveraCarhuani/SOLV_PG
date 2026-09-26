import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'tech-logo',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="tech-logo-container" [class]="'logo-' + iconType" [style.width.px]="size" [style.height.px]="size">
      @switch (iconType) {
        @case ('python') {
          <svg viewBox="0 0 24 24" width="100%" height="100%" fill="none">
            <path d="M11.92 2C6.98 2 7.29 4.14 7.29 4.14L7.3 6.36H12V7.05H5.43S2 6.66 2 11.64C2 16.63 5 16.35 5 16.35H6.43V14.18S6.27 11.64 8.86 11.64H13.43S15.86 11.75 15.86 9.38V4.47S16.29 2 11.92 2ZM9.86 3.42C10.41 3.42 10.86 3.87 10.86 4.42C10.86 4.97 10.41 5.42 9.86 5.42C9.31 5.42 8.86 4.97 8.86 4.42C8.86 3.87 9.31 3.42 9.86 3.42Z" fill="#387EB8"/>
            <path d="M12.08 22C17.02 22 16.71 19.86 16.71 19.86L16.7 17.64H12V16.95H18.57S22 17.34 22 12.36C22 7.37 19 7.65 19 7.65H17.57V9.82S17.73 12.36 15.14 12.36H10.57S8.14 12.25 8.14 14.62V19.53S7.71 22 12.08 22ZM14.14 20.58C13.59 20.58 13.14 20.13 13.14 19.58C13.14 19.03 13.59 18.58 14.14 18.58C14.69 18.58 15.14 19.03 15.14 19.58C15.14 20.13 14.69 20.58 14.14 20.58Z" fill="#FFE052"/>
          </svg>
        }
        @case ('rust') {
          <svg viewBox="0 0 24 24" width="100%" height="100%" fill="none">
            <circle cx="12" cy="12" r="10" stroke="#CE412B" stroke-width="1.8"/>
            <path d="M8.5 7H13.2C14.8 7 15.8 7.8 15.8 9.3C15.8 10.8 14.7 11.5 13.5 11.7L16 17H13.6L11.4 12.3H10.5V17H8.5V7ZM10.5 10.8H12.8C13.6 10.8 14.1 10.3 14.1 9.4C14.1 8.5 13.6 8.2 12.8 8.2H10.5V10.8Z" fill="#CE412B"/>
          </svg>
        }
        @case ('node') {
          <svg viewBox="0 0 24 24" width="100%" height="100%" fill="none">
            <path d="M12 2L21 7.2V16.8L12 22L3 16.8V7.2L12 2Z" fill="#5FA04E"/>
            <path d="M12 4.5L18.5 8.2V15.8L12 19.5L5.5 15.8V8.2L12 4.5Z" fill="#333333"/>
            <path d="M10 8.5V15.5L14 11.5V8.5H10Z" fill="#5FA04E"/>
          </svg>
        }
        @case ('go') {
          <svg viewBox="0 0 24 24" width="100%" height="100%" fill="none">
            <rect width="24" height="24" rx="6" fill="#00ACD7"/>
            <text x="12" y="16" font-size="11" font-weight="900" font-family="sans-serif" fill="#FFFFFF" text-anchor="middle">GO</text>
          </svg>
        }
        @case ('java') {
          <svg viewBox="0 0 24 24" width="100%" height="100%" fill="none">
            <path d="M7 19C10.5 19.5 15 19.5 18 19C15.5 20.5 9 20.8 6 20.5L7 19Z" fill="#E76F00"/>
            <path d="M6 16C9.5 16.5 14 16.5 17 16C14.5 17.5 8 17.8 5 17.5L6 16Z" fill="#E76F00"/>
            <path d="M11 11C11.5 9.5 11 8 9.5 6.5C8 5 9 3 9 3C9 3 10 4.5 9.5 6C9 7.5 10 8.5 11 11Z" stroke="#5382A1" stroke-width="1.5" stroke-linecap="round"/>
            <path d="M14 10C14.5 8.5 14 7.5 13 6C12 4.5 13 2 13 2C13 2 14 3.5 13.5 5C13 6.5 14 7.5 14 10Z" stroke="#E76F00" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        }
        @case ('cpp') {
          <svg viewBox="0 0 24 24" width="100%" height="100%" fill="none">
            <rect width="24" height="24" rx="6" fill="#00599C"/>
            <text x="12" y="16" font-size="10" font-weight="800" font-family="monospace" fill="#FFFFFF" text-anchor="middle">C++</text>
          </svg>
        }
        @case ('db') {
          <svg viewBox="0 0 24 24" width="100%" height="100%" fill="none">
            <ellipse cx="12" cy="6" rx="8" ry="3" fill="#38BDF8" stroke="#0284C7" stroke-width="1.5"/>
            <path d="M4 6V12C4 13.66 7.58 15 12 15C16.42 15 20 13.66 20 12V6" stroke="#0284C7" stroke-width="1.5"/>
            <path d="M4 12V18C4 19.66 7.58 21 12 21C16.42 21 20 19.66 20 18V12" stroke="#0284C7" stroke-width="1.5"/>
          </svg>
        }
        @default {
          <svg viewBox="0 0 24 24" width="100%" height="100%" fill="none">
            <rect width="24" height="24" rx="6" fill="#2496ED"/>
            <path d="M4 13H20V15C20 17.2 16.4 19 12 19C7.6 19 4 17.2 4 15V13Z" fill="#FFFFFF"/>
            <rect x="5" y="10" width="2.5" height="2" fill="#FFFFFF"/>
            <rect x="8.5" y="10" width="2.5" height="2" fill="#FFFFFF"/>
            <rect x="12" y="10" width="2.5" height="2" fill="#FFFFFF"/>
            <rect x="8.5" y="7.5" width="2.5" height="2" fill="#FFFFFF"/>
            <rect x="12" y="7.5" width="2.5" height="2" fill="#FFFFFF"/>
          </svg>
        }
      }
    </div>
  `,
  styles: [`
    .tech-logo-container {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      border-radius: 8px;
      padding: 4px;
      flex-shrink: 0;
      background-color: #F8FAFC;
      border: 1px solid var(--border-subtle, #E2E8F0);
      transition: transform 150ms ease;

      &.logo-python { background-color: #F0F7FF; border-color: #BAE6FD; }
      &.logo-rust { background-color: #FFF7ED; border-color: #FED7AA; }
      &.logo-node { background-color: #F0FDF4; border-color: #BBF7D0; }
      &.logo-go { background-color: #F0F9FF; border-color: #BAE6FD; }
      &.logo-java { background-color: #FFF7ED; border-color: #FFEDD5; }
      &.logo-cpp { background-color: #EFF6FF; border-color: #BFDBFE; }
      &.logo-db { background-color: #F0F9FF; border-color: #E0F2FE; }
      &.logo-docker { background-color: #EFF6FF; border-color: #BFDBFE; }
    }
  `]
})
export class TechLogoComponent {
  @Input({ required: true }) iconType: 'python' | 'rust' | 'node' | 'go' | 'java' | 'cpp' | 'db' | 'docker' = 'docker';
  @Input() size = 32;
}
