import { Component, signal, inject, effect, computed } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { EMPTY, Subject, switchMap, catchError } from 'rxjs';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatRadioModule } from '@angular/material/radio';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatChipsModule } from '@angular/material/chips';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { QuizService, QuizQuestion, QuizResult } from '../../services/quiz.service';

@Component({
  selector: 'app-quiz',
  standalone: true,
  imports: [
    FormsModule,
    RouterLink,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatRadioModule,
    MatCheckboxModule,
    MatChipsModule,
    MatProgressBarModule,
  ],
  templateUrl: './quiz.component.html',
  styleUrl: './quiz.component.scss'
})
export class QuizComponent {
  private route = inject(ActivatedRoute);
  private quizService = inject(QuizService);

  readonly chapterId = Number(this.route.snapshot.paramMap.get('chapterId'));
  readonly error = signal(this.chapterId > 0 ? '' : 'Ungültige Kapitel-ID.');

  readonly questions = toSignal(
    this.quizService.getQuestions(this.chapterId).pipe(
      catchError((err: Error) => { this.error.set(err.message); return EMPTY; })
    ),
    { initialValue: [] as QuizQuestion[] }
  );

  readonly answers = signal<number[][]>([]);
  readonly currentQuestionIndex = signal(0);
  readonly totalQuestions = computed(() => this.questions().length);
  readonly allAnswered = computed(() => {
    const questions = this.questions();
    const givenAnswers = this.answers();
    if (questions.length === 0) {
      return false;
    }
    return questions.every((_, index) => (givenAnswers[index] ?? []).length > 0);
  });
  readonly answeredCount = computed(() => this.answers().filter(answer => answer.length > 0).length);
  readonly progressPercent = computed(() => {
    const total = this.totalQuestions();
    if (total === 0) {
      return 0;
    }
    return (this.answeredCount() / total) * 100;
  });
  readonly currentQuestion = computed(() => {
    const questions = this.questions();
    const index = this.currentQuestionIndex();
    return questions[index];
  });

  private readonly submitTrigger$ = new Subject<number[][]>();

  readonly result = toSignal<QuizResult>(
    this.submitTrigger$.pipe(
      switchMap(answers =>
        this.quizService.submitQuiz(this.chapterId, answers).pipe(
          catchError((err: Error) => { this.error.set(err.message); return EMPTY; })
        )
      )
    )
  );

  constructor() {
    effect(() => {
      const questions = this.questions();
      this.answers.set(new Array(questions.length).fill(null).map(() => []));
      this.currentQuestionIndex.set(0);
    });
  }

  previousQuestion(): void {
    this.currentQuestionIndex.update(index => Math.max(0, index - 1));
  }

  nextQuestion(): void {
    this.currentQuestionIndex.update(index => Math.min(this.totalQuestions() - 1, index + 1));
  }

  isCurrentQuestionAnswered(): boolean {
    return this.isQuestionAnswered(this.currentQuestionIndex());
  }

  isQuestionAnswered(questionIndex: number): boolean {
    return (this.answers()[questionIndex] ?? []).length > 0;
  }

  getQuestionText(question: QuizQuestion): string {
    const candidate = question as QuizQuestion & {
      question_text?: string;
      text?: string;
    };
    const text = candidate.question ?? candidate.question_text ?? candidate.text ?? '';
    const normalized = String(text).trim();
    return normalized.length > 0 ? normalized : 'Fragetext nicht verfugbar';
  }

  toggleOption(questionIndex: number, optionIndex: number, quizType: 'single' | 'multiple'): void {
    this.answers.update(all => {
      const updated = all.map(a => [...a]);
      const current = updated[questionIndex] ?? [];
      if (quizType === 'single') {
        updated[questionIndex] = [optionIndex];
      } else {
        const idx = current.indexOf(optionIndex);
        updated[questionIndex] = idx >= 0
          ? current.filter((_, i) => i !== idx)
          : [...current, optionIndex];
      }
      return updated;
    });
  }

  isChecked(questionIndex: number, optionIndex: number): boolean {
    return (this.answers()[questionIndex] ?? []).includes(optionIndex);
  }

  isAnswerCorrectOption(correctAnswer: number[], optionIndex: number): boolean {
    return correctAnswer.includes(optionIndex);
  }

  isCorrectlySelectedOption(userAnswer: number[], correctAnswer: number[], optionIndex: number): boolean {
    return userAnswer.includes(optionIndex) && correctAnswer.includes(optionIndex);
  }

  isWronglySelectedOption(userAnswer: number[], correctAnswer: number[], optionIndex: number): boolean {
    return userAnswer.includes(optionIndex) && !correctAnswer.includes(optionIndex);
  }

  isMissedCorrectOption(userAnswer: number[], correctAnswer: number[], optionIndex: number): boolean {
    return !userAnswer.includes(optionIndex) && correctAnswer.includes(optionIndex);
  }

  submit(): void {
    if (!this.allAnswered()) {
      this.error.set('Bitte alle Fragen beantworten, bevor ausgewertet wird.');
      return;
    }
    this.error.set('');
    this.submitTrigger$.next(this.answers());
  }
}
 