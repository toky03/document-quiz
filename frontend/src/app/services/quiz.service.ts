import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface Chapter {
  id: number;
  title: string;
  source_name: string;
  question_count: number;
}

export interface QuizQuestion {
  question: string;
  quiz_type: 'single' | 'multiple';
  options: string[];
  correct_options: number[];
  answer: string;
}

export interface QuizResult {
  correct_count: number;
  total_count: number;
  results: Array<{
    index: number;
    question: string;
    user_answer: number[];
    correct_answer: number[];
    is_correct: boolean;
    options: string[];
    quiz_type: string;
  }>;
}

@Injectable({ providedIn: 'root' })
export class QuizService {
  private apiUrl = 'http://localhost:8080/api';

  constructor(private http: HttpClient) {}

  uploadFiles(files: File[], model: string, apiKey: string): Observable<unknown> {
    const formData = new FormData();
    for (const file of files) {
      formData.append('files', file, file.name);
    }
    formData.append('model', model);
    if (apiKey.trim()) {
      formData.append('api_key', apiKey.trim());
    }
    return this.http.post(`${this.apiUrl}/upload`, formData);
  }

  getChapters(): Observable<Chapter[]> {
    return this.http.get<Chapter[]>(`${this.apiUrl}/chapters`);
  }

  getQuestions(chapterId: number): Observable<QuizQuestion[]> {
    return this.http.get<QuizQuestion[]>(`${this.apiUrl}/chapters/${chapterId}/questions`);
  }

  submitQuiz(chapterId: number, answers: number[][]): Observable<QuizResult> {
    return this.http.post<QuizResult>(`${this.apiUrl}/quiz/submit`, {
      chapter_id: chapterId,
      answers
    });
  }
}
