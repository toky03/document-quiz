import { Component, inject, signal } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';

export interface QuizOptions {
  shuffleQuestions: boolean;
  shuffleOptions: boolean;
  practiceMode: boolean;
}

@Component({
  selector: 'app-quiz-options-dialog',
  standalone: true,
  imports: [MatDialogModule, MatButtonModule, MatCheckboxModule],
  template: `
    <h2 mat-dialog-title>Quiz starten</h2>
    <mat-dialog-content class="options-content">
      <div class="option-row">
        <mat-checkbox
          [checked]="shuffleQuestions()"
          (change)="shuffleQuestions.set($event.checked)"
        >
          Fragen zufällig sortieren
        </mat-checkbox>
      </div>
      <div class="option-row">
        <mat-checkbox
          [checked]="shuffleOptions()"
          (change)="shuffleOptions.set($event.checked)"
        >
          Antworten zufällig sortieren
        </mat-checkbox>
      </div>
      <div class="option-row">
        <mat-checkbox
          [checked]="practiceMode()"
          (change)="practiceMode.set($event.checked)"
        >
          Übungsmodus
        </mat-checkbox>
        <span class="option-hint">
          Antworten und Erklärungen werden direkt nach jeder Frage angezeigt; der Punktestand läuft mit.
        </span>
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-stroked-button [mat-dialog-close]="null">Abbrechen</button>
      <button mat-flat-button color="primary" (click)="start()">Starten</button>
    </mat-dialog-actions>
  `,
  styles: `
    .options-content {
      display: flex;
      flex-direction: column;
      gap: 12px;
      min-width: 320px;
    }
    .option-row {
      display: flex;
      flex-direction: column;
      gap: 4px;
    }
    .option-hint {
      font-size: 12px;
      color: rgba(0, 0, 0, 0.6);
      margin-left: 32px;
    }
  `,
})
export class QuizOptionsDialogComponent {
  private readonly data = inject<Partial<QuizOptions> | null>(MAT_DIALOG_DATA, { optional: true });
  private readonly dialogRef = inject(MatDialogRef<QuizOptionsDialogComponent, QuizOptions | null>);

  readonly shuffleQuestions = signal(this.data?.shuffleQuestions ?? true);
  readonly shuffleOptions = signal(this.data?.shuffleOptions ?? true);
  readonly practiceMode = signal(this.data?.practiceMode ?? false);

  start(): void {
    this.dialogRef.close({
      shuffleQuestions: this.shuffleQuestions(),
      shuffleOptions: this.shuffleOptions(),
      practiceMode: this.practiceMode(),
    });
  }
}
