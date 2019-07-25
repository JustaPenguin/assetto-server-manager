let gulp = require('gulp');
let browserify = require('browserify');
let source = require('vinyl-source-stream');
let tsify = require('tsify');
let sourcemaps = require('gulp-sourcemaps');
let buffer = require('vinyl-buffer');
let uglify = require('gulp-uglify-es').default;
let sass = require("gulp-sass");
let autoPrefixer = require("gulp-autoprefixer");

gulp.task('build-js', buildJS);
gulp.task("build-sass", buildSass);

gulp.task('watch', () => {
    gulp.watch(['src/**/*.ts', 'src/**/*.js'], buildJS);
    gulp.watch("./sass/**/*.scss", buildSass);
});

gulp.task("copy", function() {
    return gulp.src("./node_modules/summernote/dist/font/*")
        .pipe(gulp.dest("../static/css/font"))
    ;
});

gulp.task('build', gulp.series('build-js', 'build-sass', 'copy'));
gulp.task('default', gulp.series('build', 'watch'));

function buildJS() {
    try {
        return browserify({
            basedir: '.',
            debug: true,
            entries: ['src/main.ts'],
            cache: {},
            packageCache: {},
        })
            .plugin(tsify)
            .transform('babelify', {
                presets: ['@babel/preset-env'],
                extensions: ['.ts']
            })
            .transform({global: true}, 'browserify-shim')
            .bundle()
            .pipe(source('bundle.js'))
            .pipe(buffer())
            .pipe(sourcemaps.init({loadMaps: true}))
            .pipe(uglify())
            .pipe(sourcemaps.write('.'))
            .pipe(gulp.dest('../static/js'));

    } catch (e) {
        console.error(e);
    }
}

function buildSass() {
    return gulp.src("./sass/server-manager.scss")
        .pipe(sourcemaps.init())
        .pipe(sass({
            outputStyle: 'compressed',
            includePaths: [
                "./node_modules"
            ]
        }))
        .pipe(autoPrefixer({
            cascade: false
        }))
        .pipe(sourcemaps.write())
        .pipe(gulp.dest("../static/css/"))
    ;
}