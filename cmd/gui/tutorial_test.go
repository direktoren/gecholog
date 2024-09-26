package main

import (
	"reflect"
	"testing"
)

func Test_addRouterTutorialGL(t *testing.T) {
	type args struct {
		t tutorial
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test addRouterTutorialGL",
			args: args{
				t: tutorial{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := addRouterTutorialGL(tt.args.t); (err != nil) != tt.wantErr {
				t.Errorf("addRouterTutorialGL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_addRouterTutorialNats2file(t *testing.T) {
	type args struct {
		t tutorial
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test addRouterTutorialNats2file",
			args: args{
				t: tutorial{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := addRouterTutorialNats2file(tt.args.t); (err != nil) != tt.wantErr {
				t.Errorf("addRouterTutorialNats2file() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_tutorial_findStep(t *testing.T) {
	type fields struct {
		Steps map[string]tutorialStep
	}
	type args struct {
		step      string
		areaIndex int
		form      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    tutorialStep
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				Steps: map[string]tutorialStep{
					"208": tutorialStep{
						name:           "Step 8: Finalize",
						form:           "form.html",
						areaIndex:      GL_CONFIG_ROUTERS,
						ID:             "208",
						NextID:         "209",
						PreviousButton: "207",
						NextButton:     "",
					},
				},
			},
			args: args{
				step:      "208",
				areaIndex: GL_CONFIG_ROUTERS,
				form:      "form.html",
			},
			want: tutorialStep{
				name:           "Step 8: Finalize",
				form:           "form.html",
				areaIndex:      GL_CONFIG_ROUTERS,
				ID:             "208",
				NextID:         "209",
				PreviousButton: "207",
				NextButton:     "",
			},
			wantErr: false,
		},
		{
			name: "Success no 2",
			fields: fields{
				Steps: map[string]tutorialStep{
					"208": tutorialStep{
						name:           "Step 8: Finalize",
						form:           "form.html",
						areaIndex:      GL_CONFIG_ROUTERS,
						ID:             "208",
						NextID:         "209",
						PreviousButton: "207",
						NextButton:     "",
					},
					"209": tutorialStep{
						name:           "Step 9: Finalize",
						form:           "form.html",
						areaIndex:      GL_CONFIG_ROUTERS,
						ID:             "209",
						NextID:         "209",
						PreviousButton: "",
						NextButton:     "",
					},
				},
			},
			args: args{
				step:      "208",
				areaIndex: GL_CONFIG_ROUTERS,
				form:      "form.html",
			},
			want: tutorialStep{
				name:           "Step 8: Finalize",
				form:           "form.html",
				areaIndex:      GL_CONFIG_ROUTERS,
				ID:             "208",
				NextID:         "209",
				PreviousButton: "207",
				NextButton:     "",
			},
			wantErr: false,
		},
		{
			name: "t.Steps is nil",
			fields: fields{
				Steps: nil, // This is the error

			},
			args: args{
				step:      "208",
				areaIndex: GL_CONFIG_ROUTERS,
				form:      "form.html",
			},
			want:    tutorialStep{},
			wantErr: true,
		},
		{
			name: "step argument is empty",
			fields: fields{
				Steps: map[string]tutorialStep{
					"208": tutorialStep{
						name:           "Step 8: Finalize",
						form:           "form.html",
						areaIndex:      GL_CONFIG_ROUTERS,
						ID:             "208",
						NextID:         "209",
						PreviousButton: "207",
						NextButton:     "",
					},
				},
			},
			args: args{
				step:      "", // This is the error
				areaIndex: GL_CONFIG_ROUTERS,
				form:      "form.html",
			},
			want:    tutorialStep{},
			wantErr: true,
		},
		{
			name: "step not found",
			fields: fields{
				Steps: map[string]tutorialStep{
					"208": tutorialStep{
						name:           "Step 8: Finalize",
						form:           "form.html",
						areaIndex:      GL_CONFIG_ROUTERS,
						ID:             "208",
						NextID:         "209",
						PreviousButton: "207",
						NextButton:     "",
					},
					"209": tutorialStep{
						name:           "Step 9: Finalize",
						form:           "form.html",
						areaIndex:      GL_CONFIG_ROUTERS,
						ID:             "209",
						NextID:         "209",
						PreviousButton: "",
						NextButton:     "",
					},
				},
			},
			args: args{
				step:      "210",
				areaIndex: GL_CONFIG_ROUTERS,
				form:      "form.html",
			},
			want:    tutorialStep{},
			wantErr: true,
		},
		{
			name: "Bad areaIndex",
			fields: fields{
				Steps: map[string]tutorialStep{
					"208": tutorialStep{
						name:           "Step 8: Finalize",
						form:           "form.html",
						areaIndex:      GL_CONFIG_ROUTERS,
						ID:             "208",
						NextID:         "209",
						PreviousButton: "207",
						NextButton:     "",
					},
				},
			},
			args: args{
				step:      "208",
				areaIndex: -1, // This is the error
				form:      "form.html",
			},
			want:    tutorialStep{},
			wantErr: true,
		},
		{
			name: "Empty form",
			fields: fields{
				Steps: map[string]tutorialStep{
					"208": tutorialStep{
						name:           "Step 8: Finalize",
						form:           "form.html",
						areaIndex:      GL_CONFIG_ROUTERS,
						ID:             "208",
						NextID:         "209",
						PreviousButton: "207",
						NextButton:     "",
					},
				},
			},
			args: args{
				step:      "208",
				areaIndex: GL_CONFIG_ROUTERS,
				form:      "", // This is the error
			},
			want:    tutorialStep{},
			wantErr: true,
		},
		{
			name: "form != ts.form",
			fields: fields{
				Steps: map[string]tutorialStep{
					"208": tutorialStep{
						name:           "Step 8: Finalize",
						form:           "form.html",
						areaIndex:      GL_CONFIG_ROUTERS,
						ID:             "208",
						NextID:         "209",
						PreviousButton: "207",
						NextButton:     "",
					},
				},
			},
			args: args{
				step:      "208",
				areaIndex: GL_CONFIG_ROUTERS,
				form:      "publish.html", // This is the error
			},
			want:    tutorialStep{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := tutorial{
				Steps: tt.fields.Steps,
			}
			got, err := tr.findStep(tt.args.step, tt.args.areaIndex, tt.args.form)
			if (err != nil) != tt.wantErr {
				t.Errorf("tutorial.findStep() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tutorial.findStep() = %v, want %v", got, tt.want)
			}
		})
	}
}
