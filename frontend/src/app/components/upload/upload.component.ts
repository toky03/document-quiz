import { Component, computed, signal, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { EMPTY, Subject, switchMap, catchError, map, startWith, tap } from 'rxjs';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { QuizService, Chapter } from '../../services/quiz.service';

interface UploadParams { files: File[]; model: string; apiKey: string; }

@Component({
  selector: 'app-upload',
  standalone: true,
  imports: [
    FormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatListModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './upload.component.html',
  styleUrl: './upload.component.scss'
})
export class UploadComponent {
  private quizService = inject(QuizService);
  private router = inject(Router);

  readonly files = signal<File[]>([]);
  readonly error = signal('');
  readonly isLoading = signal(false);

  model = 'gpt-4.1-mini';
  apiKey = '';

  private readonly refreshTrigger$ = new Subject<void>();
  private readonly uploadTrigger$ = new Subject<UploadParams>();

  readonly chapters = toSignal(
    this.refreshTrigger$.pipe(
      startWith(null),
      switchMap(() =>
        this.quizService.getChapters().pipe(
          map((chapters) => Array.isArray(chapters) ? chapters : []),
          catchError(() => EMPTY)
        )
      )
    ),
    { initialValue: [] as Chapter[] }
  );

  readonly chapterList = computed(() => this.chapters() ?? []);

  private readonly _uploadEffect = toSignal(
    this.uploadTrigger$.pipe(
      tap(() => { this.isLoading.set(true); this.error.set(''); }),
      switchMap(({ files, model, apiKey }) =>
        this.quizService.uploadFiles(files, model, apiKey).pipe(
          tap(() => {
            this.isLoading.set(false);
            this.files.set([]);
            this.refreshTrigger$.next();
          }),
          catchError((err: Error) => {
            this.isLoading.set(false);
            this.error.set(err.message);
            return EMPTY;
          })
        )
      )
    )
  );

  onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.files.set(input.files ? Array.from(input.files) : []);
  }

  upload(): void {
    if (this.files().length === 0) {
      this.error.set('Bitte mindestens eine PDF-Datei auswählen.');
      return;
    }
    this.uploadTrigger$.next({ files: this.files(), model: this.model, apiKey: this.apiKey });
  }

  openQuiz(chapter: Chapter): void {
    this.router.navigate(['/quiz', chapter.id]);
  }
}
