import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { LucideCheckCircle, LucideSend, LucideCopy } from '@lucide/angular';

export interface TokenFeedbackData {
  title: string;
  email: string;
  inviteUrl: string;
}

@Component({
  selector: 'teacher-invite-feedback',
  standalone: true,
  imports: [CommonModule, FormsModule, ModalShellComponent, LucideCheckCircle, LucideSend, LucideCopy],
  templateUrl: './teacher-invite-feedback.component.html',
  styleUrl: './teacher-invite-feedback.component.scss'
})
export class TeacherInviteFeedbackComponent {
  readonly feedback = input.required<TokenFeedbackData | null>();
  readonly copied = input.required<boolean>();

  readonly dismiss = output<void>();
  readonly copyTokenUrl = output<string>();
}
