import { Component, inject } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';

export interface ConfirmDialogData {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
}

@Component({
  selector: 'app-confirm-dialog',
  standalone: true,
  imports: [MatDialogModule, MatButtonModule],
  template: `
    <h2 mat-dialog-title>{{ data.title }}</h2>
    <mat-dialog-content>{{ data.message }}</mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-stroked-button [mat-dialog-close]="false">
        {{ data.cancelLabel ?? 'Abbrechen' }}
      </button>
      <button mat-flat-button class="btn-warn" [mat-dialog-close]="true">
        {{ data.confirmLabel ?? 'Löschen' }}
      </button>
    </mat-dialog-actions>
  `,
    styles: `
    .btn-warn {
      --mat-button-filled-container-color: #c20427;
      --mat-button-filled-label-text-color: #ffffff;
    }
  `
})
export class ConfirmDialogComponent {
  readonly data = inject<ConfirmDialogData>(MAT_DIALOG_DATA);
  readonly dialogRef = inject(MatDialogRef<ConfirmDialogComponent>);
}
