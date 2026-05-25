import { Component, computed, signal, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { EMPTY, Subject, switchMap, catchError, map, startWith, tap, filter } from 'rxjs';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSelectModule } from '@angular/material/select';
import { MatDialog } from '@angular/material/dialog';
import { QuizService, Chapter } from '../../services/quiz.service';
import { ConfirmDialogComponent } from '../confirm-dialog/confirm-dialog.component';

type Provider = 'openai' | 'claude_cli';

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
    MatSelectModule,
  ],
  templateUrl: './upload.component.html',
  styleUrl: './upload.component.scss'
})
export class UploadComponent {

  private quizService = inject(QuizService);
  private router = inject(Router);
  private dialog = inject(MatDialog);

  readonly files = signal<File[]>([]);
  readonly error = signal('');
  readonly isLoading = signal(false);

  model = 'gpt-4.1-mini';
  apiKey = '';
  readonly provider = signal<Provider>('openai');

  private readonly refreshTrigger$ = new Subject<void>();
  private readonly uploadTrigger$ = new Subject<UploadParams>();
  private readonly apiKeyRefreshTrigger$ = new Subject<void>();
  private readonly providerRefreshTrigger$ = new Subject<void>();

  readonly isOpenAI = computed(() => this.provider() === 'openai');

  readonly hasSavedApiKey = toSignal(
    this.apiKeyRefreshTrigger$.pipe(
      startWith(null),
      switchMap(() =>
        this.quizService.hasSavedApiKey().pipe(
          catchError(() => EMPTY)
        )
      )
    ),
    { initialValue: false }
  );

  private readonly _providerEffect = toSignal(
    this.providerRefreshTrigger$.pipe(
      startWith(null),
      switchMap(() =>
        this.quizService.getProvider().pipe(
          tap(p => {
            const next = (p === 'claude_cli' ? 'claude_cli' : 'openai') as Provider;
            this.provider.set(next);
            if (next === 'claude_cli' && (this.model === '' || this.model.startsWith('gpt-'))) {
              this.model = 'sonnet';
            }
          }),
          catchError(() => EMPTY)
        )
      )
    )
  );

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

  clearApiKey(): void {
    this.quizService.clearApiKey().pipe(
      catchError((err: Error) => {
        this.error.set(err.message);
        return EMPTY;
      })
    ).subscribe(() => {
      this.apiKeyRefreshTrigger$.next();
    });
  }

  onProviderChange(next: Provider): void {
    this.quizService.setProvider(next).pipe(
      catchError((err: Error) => {
        this.error.set(err.message);
        return EMPTY;
      })
    ).subscribe(() => {
      this.provider.set(next);
      if (next === 'claude_cli' && (this.model === '' || this.model.startsWith('gpt-'))) {
        this.model = 'sonnet';
      }
      if (next === 'openai' && !this.model.startsWith('gpt-')) {
        this.model = 'gpt-4.1-mini';
      }
    });
  }

  deleteChapter(chapter: Chapter): void {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: 'Kapitel löschen',
        message: `Soll das Kapitel "${chapter.title}" wirklich gelöscht werden?`,
      },
    });

    dialogRef.afterClosed().pipe(
      filter((confirmed: boolean) => confirmed),
      switchMap(() =>
        this.quizService.deleteChapter(chapter.id).pipe(
          catchError((err: Error) => {
            this.error.set(err.message);
            return EMPTY;
          })
        )
      )
    ).subscribe(() => {
      this.refreshTrigger$.next();
    });
  }
}
