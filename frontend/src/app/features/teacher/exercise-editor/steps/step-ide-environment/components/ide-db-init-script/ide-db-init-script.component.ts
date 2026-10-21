import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideDatabase } from '@lucide/angular';

@Component({
  selector: 'ide-db-init-script',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideDatabase
  ],
  templateUrl: './ide-db-init-script.component.html',
  styleUrl: './ide-db-init-script.component.scss'
})
export class IdeDbInitScriptComponent {
  dbInitScript = input<string>('');
  hasDatabaseService = input<boolean>(false);

  initScriptChange = output<string>();
}
